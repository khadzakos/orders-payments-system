package kafka_infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type MessageHandler func(ctx context.Context, message kafka.Message) error

type Consumer struct {
	reader  *kafka.Reader
	logger  *zap.Logger
	handler MessageHandler
}

func NewConsumer(brokers []string, topic, groupID string, handler MessageHandler, l *zap.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		Logger:         kafka.LoggerFunc(l.Sugar().Infof),
		ErrorLogger:    kafka.LoggerFunc(l.Sugar().Errorf),
	})

	return &Consumer{
		reader:  reader,
		logger:  l,
		handler: handler,
	}

	return nil
}

func (c *Consumer) Consume(ctx context.Context) error {
	c.logger.Info("Kafka consumer starting message consumption",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer.", zap.String("topic", c.reader.Config().Topic))
			return ctx.Err()
		default:
		}

		fetchCtx, cancelFetch := context.WithTimeout(ctx, 5*time.Second) // Use parent ctx for cancellation
		m, err := c.reader.FetchMessage(fetchCtx)
		cancelFetch()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, kafka.ErrGroupClosed) {
				c.logger.Info("Consumer stopping due to context cancellation or reader closure.", zap.Error(err), zap.String("topic", c.reader.Config().Topic))
				return nil
			}
			c.logger.Error("Error fetching message from Kafka", zap.Error(err), zap.String("topic", c.reader.Config().Topic))
			time.Sleep(1 * time.Second)
			continue
		}

		handleCtx, cancelHandler := context.WithTimeout(context.Background(), 25*time.Second)
		if err := c.handler(handleCtx, m); err != nil {
			c.logger.Error("Error handling Kafka message",
				zap.String("topic", m.Topic),
				zap.Int("partition", m.Partition),
				zap.Int64("offset", m.Offset),
				zap.Error(err))
			cancelHandler()
			continue
		}
		cancelHandler()

		commitCtx, cancelCommit := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.reader.CommitMessages(commitCtx, m); err != nil {
			c.logger.Error("Failed to commit offset for message",
				zap.String("topic", m.Topic),
				zap.Int("partition", m.Partition),
				zap.Int64("offset", m.Offset),
				zap.Error(err))
		}
		cancelCommit()
	}
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		c.logger.Error("Failed to close Kafka consumer reader", zap.Error(err), zap.String("topic", c.reader.Config().Topic))
		return fmt.Errorf("failed to close Kafka consumer reader: %w", err)
	}
	c.logger.Info("Kafka consumer reader closed.", zap.String("topic", c.reader.Config().Topic))
	return nil
}
