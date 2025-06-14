package kafka_infra

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer interface {
	Produce(ctx context.Context, key, topic string, value []byte) error
	Close() error
}

type kafkaProducer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

func NewProducer(brokerURLs []string, defaultTopic string, logger *zap.Logger) Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerURLs...),
		Topic:        defaultTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  3,
		Async:        true,
		Completion:   nil,
		Logger:       kafka.LoggerFunc(func(msg string, args ...interface{}) { logger.Debug(fmt.Sprintf(msg, args...)) }),
		ErrorLogger:  kafka.LoggerFunc(func(msg string, args ...interface{}) { logger.Error(fmt.Sprintf(msg, args...)) }),
	}

	writer.Completion = func(messages []kafka.Message, err error) {
		for _, msg := range messages {
			if err != nil {
				logger.Error("Failed to write message to Kafka",
					zap.String("topic", msg.Topic),
					zap.String("key", string(msg.Key)),
					zap.Error(err),
				)
				continue
			}
			logger.Debug("Message written to Kafka successfully",
				zap.String("topic", msg.Topic),
				zap.String("key", string(msg.Key)),
			)
		}
	}

	return &kafkaProducer{
		writer: writer,
		logger: logger,
	}
}

func (p *kafkaProducer) Produce(ctx context.Context, key, topic string, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	produceCtx, cancel := context.WithTimeout(ctx, p.writer.WriteTimeout)
	defer cancel()

	err := p.writer.WriteMessages(produceCtx, msg)
	if err != nil {
		p.logger.Error("Failed to produce message to Kafka",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to produce message to Kafka: %w", err)
	}
	p.logger.Debug("Message produced to Kafka successfully",
		zap.String("topic", topic),
		zap.String("key", key),
	)
	return nil
}

func (p *kafkaProducer) Close() error {
	if p.writer == nil {
		return nil
	}
	err := p.writer.Close()
	if err != nil {
		p.logger.Error("Failed to close Kafka producer", zap.Error(err))
		return fmt.Errorf("failed to close Kafka producer: %w", err)
	}
	p.logger.Info("Kafka Producer closed.")
	return nil
}
