package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"payments/internal/app/payments"
	"payments/internal/config"
	payments_http "payments/internal/handler/http/payments"
	kafka_handler "payments/internal/handler/kafka"
	"payments/internal/infrastucture/database"
	kafka_infra "payments/internal/infrastucture/kafka"
	"payments/internal/outbox"
	"payments/internal/repository/accounts_repo"
	"payments/internal/repository/inbox_repo"
	"payments/internal/repository/outbox_repo"
	"payments/internal/repository/payments_repo"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ensureKafkaTopics(ctx context.Context, brokerURLs []string, topics []string, logger *zap.Logger) error {
	conn, err := kafka.DialContext(ctx, "tcp", brokerURLs[0])
	if err != nil {
		return fmt.Errorf("failed to dial kafka broker for admin operations: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get kafka controller: %w", err)
	}
	controllerConn, err := kafka.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to dial kafka controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := make([]kafka.TopicConfig, len(topics))
	for i, topic := range topics {
		topicConfigs[i] = kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		if err == kafka.TopicAlreadyExists {
			logger.Info("One or more Kafka topics already exist, skipping creation.")
		} else {
			return fmt.Errorf("failed to create Kafka topics: %w", err)
		}
	} else {
		logger.Info("Kafka topics ensured successfully.", zap.Strings("topics", topics))
	}

	return nil
}

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
	migrateDSN := cfg.GetDBMigrationConnectionString()
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

	kafkaBrokers := cfg.GetKafkaBrokers()
	requiredTopics := []string{
		cfg.KafkaOrderEventsTopic,
		cfg.KafkaPaymentStatusTopic,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = ensureKafkaTopics(ctx, kafkaBrokers, requiredTopics, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to ensure Kafka topics", zap.Error(err))
	}

	accountRepository := accounts_repo.NewAccountRepository(db)
	inboxRepository := inbox_repo.NewInboxRepository(db)
	outboxRepository := outbox_repo.NewOutboxRepository(db)
	paymentRepository := payments_repo.NewPaymentRepository(db)

	paymentService := payments.NewPaymentService(
		db,
		accountRepository,
		paymentRepository,
		inboxRepository,
		outboxRepository,
		appLogger.With(zap.String("component", "PaymentService")),
	)
	appLogger.Info("Payment Service initialized.")

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	payments_http.RegisterRoutes(router, paymentService, appLogger.With(zap.String("component", "HTTPHandler")))

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", 8082),
		Handler: router,
	}
	appLogger.Info("HTTP server configured.")

	kafkaProducer := kafka_infra.NewProducer(
		cfg.GetKafkaBrokers(),
		cfg.KafkaPaymentStatusTopic,
		appLogger.With(zap.String("component", "KafkaProducer")),
	)
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			appLogger.Error("Error closing Kafka producer", zap.Error(err))
		} else {
			appLogger.Info("Kafka producer closed.")
		}
	}()
	appLogger.Info("Kafka producer created successfully.")

	outboxProcessor := outbox.NewProcessor(
		db,
		outboxRepository,
		kafkaProducer,
		cfg.KafkaPaymentStatusTopic, // Added topic
		cfg.OutboxPollInterval,
		cfg.OutboxPollTimeout,
		appLogger.With(zap.String("component", "OutboxProcessor")),
	)
	appLogger.Info("Outbox Processor initialized.")

	// Kafka Consumer for Order Events
	orderCreatedHandler := kafka_handler.OrderCreatedMessageHandler(
		paymentService,
		appLogger.With(zap.String("component", "OrderCreatedHandler")),
	)

	orderEventsConsumer := kafka_infra.NewConsumer(
		cfg.GetKafkaBrokers(),
		cfg.KafkaOrderEventsTopic,
		"payments-order-events-group", // Consumer group ID
		orderCreatedHandler,           // Pass the handler function directly
		appLogger.With(zap.String("component", "OrderEventsConsumer")),
	)
	appLogger.Info("Order Events Kafka Consumer initialized.")

	ctxMain, cancelMain := context.WithCancel(context.Background())
	go func() {
		appLogger.Info("Starting HTTP server", zap.String("address", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Запуск Outbox Processor
	go func() {
		appLogger.Info("Starting Outbox Processor...")
		outboxProcessor.Start(ctxMain)
		appLogger.Info("Outbox Processor stopped.")
	}()

	// Start Kafka Consumer for Order Events
	go func() {
		appLogger.Info("Starting Order Events Kafka Consumer...")
		if err := orderEventsConsumer.Consume(ctxMain); err != nil {
			// Log error unless it's due to context cancellation (graceful shutdown)
			if err != context.Canceled && err != context.DeadlineExceeded && err != kafka.ErrGroupClosed {
				appLogger.Error("Order Events Kafka Consumer failed", zap.Error(err))
			}
		}
		appLogger.Info("Order Events Kafka Consumer stopped.")
	}()

	// --- Graceful Shutdown ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM) // Ловим Ctrl+C или SIGTERM (от Docker/Kubernetes)

	<-sigChan // Блокируем main горутину до получения сигнала
	appLogger.Info("Shutting down application...")

	// Отменяем главный контекст, чтобы все горутины начали завершаться
	cancelMain()

	// Даем горутинам время на завершение
	// Создаем контекст с таймаутом для шатдауна HTTP-сервера и других очисток
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// 1. Завершаем HTTP-сервер
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("HTTP server graceful shutdown failed", zap.Error(err))
	} else {
		appLogger.Info("HTTP server gracefully shut down.")
	}

	// 2. Завершаем Kafka Consumer (он должен завершиться через ctxMain)
	// Добавляем небольшой таймаут, чтобы дождаться завершения горутины консьюмера
	// The consumer's Consume method will return when ctxMain is canceled.
	// We also explicitly close it to release resources if it has a Close method.
	if err := orderEventsConsumer.Close(); err != nil {
		appLogger.Error("Error closing Order Events Kafka Consumer", zap.Error(err))
	} else {
		appLogger.Info("Order Events Kafka Consumer closed.")
	}
	// Wait for the consumer goroutine to finish, relying on ctxMain cancellation
	// This select block might be redundant if Close() is blocking or if ctxMain.Done() is sufficient
	select {
	case <-time.After(2 * time.Second): // Shorter timeout as Close() should be quick or ctxMain handles it
		appLogger.Warn("Order Events Kafka Consumer goroutine might not have fully stopped after Close().")
	case <-ctxMain.Done():
		appLogger.Info("Order Events Kafka Consumer goroutine confirmed stopped via context.")
	}

	// 3. Завершаем Outbox Processor (он также должен завершиться через ctxMain)
	// Аналогично консьюмеру, даем ему время завершиться.
	// OutboxProcessor.Stop() не нужен, если Start() слушает ctx.Done()
	select {
	case <-time.After(5 * time.Second):
		appLogger.Warn("Outbox Processor did not stop cleanly within 5 seconds.")
	case <-ctxMain.Done():
	}
	// Если бы в Outbox Processor был отдельный метод Stop(), вызывали бы его здесь.
	// Поскольку он слушает ctxMain.Done(), это достаточно.

	appLogger.Info("Application gracefully shut down.")

}
