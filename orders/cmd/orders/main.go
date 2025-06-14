package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"orders/internal/app/orders"
	"orders/internal/config"
	http_orders "orders/internal/handler/http/orders"
	kafka_handler "orders/internal/handler/kafka"
	"orders/internal/infrastructure/database"
	"orders/internal/infrastructure/kafka"
	postgres_order_repo "orders/internal/repository/order_repo/postgres"
	postgres_outbox_repo "orders/internal/repository/outbox_repo/postgres"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.TimeKey = "timestamp"

	appLogger, err := zapConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create zap logger: %v\n", err)
		os.Exit(1)
	}
	appLogger.Info("Order Service starting...")

	appLogger.Info("Waiting for database to be available...")
	dbConfig := database.DBConfig{
		Host:     cfg.DBConfig.DBHost,
		Port:     cfg.DBConfig.DBPort,
		User:     cfg.DBConfig.DBUser,
		Password: cfg.DBConfig.DBPassword,
		DBName:   cfg.DBConfig.DBName,
		SSLMode:  cfg.DBConfig.DBSSLMode,
	}

	var db *sql.DB
	maxRetries := 10
	retryDelay := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		db, err = database.NewPostgresDB(dbConfig)
		if err == nil {
			appLogger.Info("Successfully connected to PostgreSQL database!")
			break
		}
		appLogger.Warn(fmt.Sprintf("Failed to connect to database (attempt %d/%d): %v. Retrying in %s...", i+1, maxRetries, err, retryDelay))
		time.Sleep(retryDelay)
	}

	if db == nil {
		appLogger.Fatal("Could not connect to database after multiple retries. Exiting.", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			appLogger.Error("Error closing database connection", zap.Error(err))
		} else {
			appLogger.Info("Database connection closed.")
		}
	}()

	appLogger.Info("Running database migrations...")
	migrateDSN := "postgres://" + cfg.GetDBMigrationConnectionString()
	m, err := migrate.New(
		"file:///app/migrations",
		migrateDSN,
	)
	if err != nil {
		appLogger.Fatal("Failed to create migrate instance", zap.Error(err))
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		appLogger.Fatal("Failed to run database migrations", zap.Error(err))
	}
	appLogger.Info("Database migrations completed successfully (or no new migrations).")

	kafkaProducer, err := kafka.NewProducer(cfg.GetKafkaBrokers(), appLogger)
	if err != nil {
		appLogger.Fatal("Failed to create Kafka producer", zap.Error(err))
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			appLogger.Error("Error closing Kafka producer", zap.Error(err))
		} else {
			appLogger.Info("Kafka producer closed.")
		}
	}()
	appLogger.Info("Kafka producer created successfully.")

	orderRepository := postgres_order_repo.NewOrderRepository(db, appLogger)
	outboxRepository := postgres_outbox_repo.NewOutboxRepository(db, appLogger)

	orderService := orders.NewOrderService(orderRepository, outboxRepository, kafkaProducer, appLogger)

	go func() {
		ticker := time.NewTicker(cfg.OutboxPollInterval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), cfg.OutboxPollTimeout)
			if err := orderService.ProcessOutbox(ctx); err != nil {
				appLogger.Error("Error processing outbox", zap.Error(err))
			}
			cancel()
		}
	}()
	appLogger.Info("Transactional Outbox sender started.")

	paymentStatusConsumerHandler := kafka_handler.NewPaymentStatusConsumer(orderService, appLogger)
	go func() {
		err := kafka.StartConsumer(
			cfg.GetKafkaBrokers(),
			cfg.KafkaPaymentStatusTopic,
			cfg.KafkaConsumerGroup,
			paymentStatusConsumerHandler.HandleMessage,
			appLogger,
		)
		if err != nil {
			appLogger.Fatal("Kafka payment status consumer failed", zap.Error(err))
		}
	}()
	appLogger.Info("Kafka payment status consumer started!")

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	http_orders.RegisterRoutes(r, orderService, appLogger.With(zap.String("component", "OrderHTTPHandler")))

	serverAddr := fmt.Sprintf(":%d", 8081)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()
	appLogger.Info("Order Service запущен", zap.String("address", serverAddr))

	<-sigChan

	appLogger.Info("Выполняется остановка Order Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		appLogger.Fatal("Order Service graceful shutdown failed", zap.Error(err))
	}
	appLogger.Info("Order Service остановлен.")
}
