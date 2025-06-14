package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DBConfig struct {
		Host     string `env:"PAYMENTS_DB_HOST"`
		Port     int    `env:"PAYMENTS_DB_PORT"`
		User     string `env:"PAYMENTS_DB_USER"`
		Password string `env:"PAYMENTS_DB_PASSWORD"`
		Name     string `env:"PAYMENTS_DB_NAME"`
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

	cfg.DBConfig.Host = getEnvOrDefault("PAYMENTS_DB_HOST", "localhost")
	cfg.DBConfig.Port = getEnvAsInt("PAYMENTS_DB_PORT", 5432)
	cfg.DBConfig.User = getEnvOrDefault("PAYMENTS_DB_USER", "user")
	cfg.DBConfig.Password = getEnvOrDefault("PAYMENTS_DB_PASSWORD", "password")
	cfg.DBConfig.Name = getEnvOrDefault("PAYMENTS_DB_NAME", "payments_db")

	cfg.KafkaBrokerURL = getEnvOrDefault("KAFKA_BROKER_URL", "localhost:9092")
	cfg.KafkaOrderEventsTopic = getEnvOrDefault("KAFKA_ORDER_EVENTS_TOPIC", "order_payment_tasks")
	cfg.KafkaPaymentStatusTopic = getEnvOrDefault("KAFKA_PAYMENT_STATUS_TOPIC", "payment_status_updates")
	cfg.KafkaConsumerGroup = getEnvOrDefault("KAFKA_CONSUMER_GROUP", "payments-service-group")

	cfg.OutboxPollInterval = getEnvAsDuration("OUTBOX_POLL_INTERVAL", 1*time.Second)
	cfg.OutboxPollTimeout = getEnvAsDuration("OUTBOX_POLL_TIMEOUT", 500*time.Millisecond)

	return cfg, nil
}

func (c *Config) GetDBConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.DBConfig.Host, c.DBConfig.Port, c.DBConfig.User, c.DBConfig.Password, c.DBConfig.Name)
}

func (c *Config) GetDBMigrationConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBConfig.User, c.DBConfig.Password, c.DBConfig.Host, c.DBConfig.Port, c.DBConfig.Name)
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

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnvOrDefault(key, strconv.Itoa(defaultValue))
	if value, err := strconv.Atoi(valueStr); err == nil {
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
