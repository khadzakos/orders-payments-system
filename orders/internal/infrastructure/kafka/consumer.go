package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type MessageHandler func(ctx context.Context, message []byte) error

func StartConsumer(brokers []string, topic, groupID string, handler MessageHandler, l *zap.Logger) error { // Изменен тип на *zap.Logger
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		Logger:         zap.NewStdLog(l.With(zap.String("kafka_component", "consumer"))),
	})

	l.Info("Kafka consumer started",
		zap.String("topic", topic),
		zap.String("group_id", groupID),
		zap.Strings("brokers", brokers))

	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			m, err := reader.FetchMessage(ctx)
			cancel()

			if err != nil {
				l.Error("Error fetching message from Kafka", zap.Error(err))
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					time.Sleep(1 * time.Second)
					continue
				}
				l.Fatal("Critical Kafka consumer error", zap.Error(err))
				return
			}

			if err := handler(context.Background(), m.Value); err != nil {
				l.Error("Error handling Kafka message",
					zap.String("topic", m.Topic),
					zap.Int("partition", m.Partition),
					zap.Int64("offset", m.Offset),
					zap.Error(err))
				continue
			}

			if err := reader.CommitMessages(ctx, m); err != nil {
				l.Error("Failed to commit offset for message",
					zap.String("topic", m.Topic),
					zap.Int("partition", m.Partition),
					zap.Int64("offset", m.Offset),
					zap.Error(err))
			} else {
				l.Debug("Committed message offset",
					zap.String("topic", m.Topic),
					zap.Int("partition", m.Partition),
					zap.Int64("offset", m.Offset))
			}
		}
	}()
	return nil
}

func CloseConsumer(reader *kafka.Reader, l *zap.Logger) error { // Изменен тип на *zap.Logger
	if err := reader.Close(); err != nil {
		l.Error("Failed to close Kafka consumer", zap.Error(err))
		return fmt.Errorf("failed to close Kafka consumer: %w", err)
	}
	l.Info("Kafka consumer closed.")
	return nil
}
