package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer interface {
	Produce(ctx context.Context, topic string, message []byte) error
	Close() error
}

type kafkaProducer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

func NewProducer(brokers []string, l *zap.Logger) (Producer, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
		Logger:   zap.NewStdLog(l.With(zap.String("kafka_component", "producer"))),
	}

	l.Info("Kafka producer initialized", zap.Strings("brokers", brokers))
	return &kafkaProducer{writer: writer, logger: l}, nil
}

func (p *kafkaProducer) Produce(ctx context.Context, topic string, message []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Value: message,
	}
	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.Error("Failed to produce message to Kafka topic",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("failed to produce message: %w", err)
	}
	p.logger.Debug("Produced message to topic", zap.String("topic", topic))
	return nil
}

func (p *kafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		p.logger.Error("Failed to close Kafka producer", zap.Error(err))
		return fmt.Errorf("failed to close Kafka producer: %w", err)
	}
	p.logger.Info("Kafka producer closed.")
	return nil
}
