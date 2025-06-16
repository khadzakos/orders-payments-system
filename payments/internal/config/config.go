package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	DBConfig struct {
		DBHost     string `env:"ORDERS_DB_HOST"`
		DBPort     string `env:"ORDERS_DB_PORT"`
		DBUser     string `env:"ORDERS_DB_USER"`
		DBPassword string `env:"ORDERS_DB_PASSWORD"`
		DBName     string `env:"ORDERS_DB_NAME"`
		DBSSLMode  string `env:"ORDERS_DB_SSLMODE"`
	}

	KafkaBrokerURL          string `env:"KAFKA_BROKER_URL"`
	KafkaOrderEventsTopic   string `env:"KAFKA_ORDER_EVENTS_TOPIC"`
	KafkaPaymentStatusTopic string `env:"KAFKA_PAYMENT_STATUS_TOPIC"`
	KafkaConsumerGroup      string `env:"KAFKA_CONSUMER_GROUP"`

	OutboxPollInterval time.Duration `env:"OUTBOX_POLL_INTERVAL"`
	OutboxPollTimeout  time.Duration `env:"OUTBOX_POLL_TIMEOUT"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	cfg.DBConfig.DBHost = getEnvOrDefault("PAYMENTS_DB_HOST", "localhost")
	cfg.DBConfig.DBPort = getEnvOrDefault("PAYMENTS_DB_PORT", "5432")
	cfg.DBConfig.DBUser = getEnvOrDefault("PAYMENTS_DB_USER", "user")
	cfg.DBConfig.DBPassword = getEnvOrDefault("PAYMENTS_DB_PASSWORD", "password")
	cfg.DBConfig.DBName = getEnvOrDefault("PAYMENTS_DB_NAME", "payments_db")
	cfg.DBConfig.DBSSLMode = getEnvOrDefault("PAYMENTS_DB_SSLMODE", "disable")

	cfg.KafkaBrokerURL = getEnvOrDefault("KAFKA_BROKER_URL", "localhost:9092")
	cfg.KafkaOrderEventsTopic = getEnvOrDefault("KAFKA_ORDER_EVENTS_TOPIC", "order_payment_tasks")
	cfg.KafkaPaymentStatusTopic = getEnvOrDefault("KAFKA_PAYMENT_STATUS_TOPIC", "payment_status_updates")
	cfg.KafkaConsumerGroup = getEnvOrDefault("KAFKA_CONSUMER_GROUP", "payments-service-group")

	cfg.OutboxPollInterval = getEnvAsDuration("OUTBOX_POLL_INTERVAL", 1*time.Second)
	cfg.OutboxPollTimeout = getEnvAsDuration("OUTBOX_POLL_TIMEOUT", 500*time.Millisecond)

	return cfg, nil
}

func (c *Config) GetDBConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBConfig.DBHost, c.DBConfig.DBPort, c.DBConfig.DBUser, c.DBConfig.DBPassword, c.DBConfig.DBName, c.DBConfig.DBSSLMode)
}

func (c *Config) GetDBMigrationConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBConfig.DBUser, c.DBConfig.DBPassword, c.DBConfig.DBHost, c.DBConfig.DBPort, c.DBConfig.DBName, c.DBConfig.DBSSLMode)
}

func (c *Config) GetKafkaBrokers() []string {
	return strings.Split(c.KafkaBrokerURL, ",")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnvOrDefault(key, defaultValue.String())
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}
