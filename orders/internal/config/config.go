package config

import (
	"fmt"
	"os"
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

	KafkaURL                string `env:"KAFKA_BROKER_URL"`
	KafkaPaymentStatusTopic string `env:"KAFKA_PAYMENT_STATUS_TOPIC"`
	KafkaConsumerGroup      string `env:"KAFKA_CONSUMER_GROUP"`

	OutboxPollInterval time.Duration `env:"OUTBOX_POLL_INTERVAL"`
	OutboxPollTimeout  time.Duration `env:"OUTBOX_POLL_TIMEOUT"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	cfg.DBConfig.DBHost = getEnvOrDefault("ORDERS_DB_HOST", "localhost")
	cfg.DBConfig.DBPort = getEnvOrDefault("ORDERS_DB_PORT", "5432")
	cfg.DBConfig.DBUser = getEnvOrDefault("ORDERS_DB_USER", "postgres")
	cfg.DBConfig.DBPassword = getEnvOrDefault("ORDERS_DB_PASSWORD", "postgres")
	cfg.DBConfig.DBName = getEnvOrDefault("ORDERS_DB_NAME", "orders_db")
	cfg.DBConfig.DBSSLMode = getEnvOrDefault("ORDERS_DB_SSLMODE", "disable")

	cfg.KafkaURL = getEnvOrDefault("KAFKA_BROKER_URL", "localhost:9092")

	cfg.KafkaPaymentStatusTopic = getEnvOrDefault("KAFKA_PAYMENT_STATUS_TOPIC", "payment_status_updates")
	cfg.KafkaConsumerGroup = getEnvOrDefault("KAFKA_CONSUMER_GROUP", "order-service-group")

	outboxPollIntervalStr := getEnvOrDefault("OUTBOX_POLL_INTERVAL", "5s")
	interval, err := time.ParseDuration(outboxPollIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid OUTBOX_POLL_INTERVAL: %w", err)
	}
	cfg.OutboxPollInterval = interval

	outboxPollTimeoutStr := getEnvOrDefault("OUTBOX_POLL_TIMEOUT", "10s")
	timeout, err := time.ParseDuration(outboxPollTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid OUTBOX_POLL_TIMEOUT: %w", err)
	}
	cfg.OutboxPollTimeout = timeout

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func (c *Config) GetDBConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBConfig.DBHost, c.DBConfig.DBPort, c.DBConfig.DBUser, c.DBConfig.DBPassword, c.DBConfig.DBName, c.DBConfig.DBSSLMode)
}

func (c *Config) GetDBMigrationConnectionString() string {
	return fmt.Sprintf("%s:%s@%s:%s/%s?sslmode=%s",
		c.DBConfig.DBUser, c.DBConfig.DBPassword, c.DBConfig.DBHost, c.DBConfig.DBPort, c.DBConfig.DBName, c.DBConfig.DBSSLMode)
}

func (c *Config) GetKafkaBrokerURL() string {
	return c.KafkaURL
}

func (c *Config) GetKafkaBrokers() []string {
	return []string{c.KafkaURL}
}
