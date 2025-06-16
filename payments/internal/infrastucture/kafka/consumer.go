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

	return nil // This error is for the goroutine, not NewConsumer itself
}

// Consume starts the consumer loop. It blocks until the context is canceled or a fatal error occurs.
func (c *Consumer) Consume(ctx context.Context) error {
	c.logger.Info("Kafka consumer starting message consumption",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)

	for {
		// Check if context is cancelled before fetching message
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer.", zap.String("topic", c.reader.Config().Topic))
			return ctx.Err()
		default:
		}

		// Use a shorter timeout for FetchMessage to allow more frequent checks of the parent context.
		fetchCtx, cancelFetch := context.WithTimeout(ctx, 5*time.Second) // Use parent ctx for cancellation
		m, err := c.reader.FetchMessage(fetchCtx)
		cancelFetch()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) { // Timeout, just continue to check parent context
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, kafka.ErrGroupClosed) { // Parent context cancelled or reader closed
				c.logger.Info("Consumer stopping due to context cancellation or reader closure.", zap.Error(err), zap.String("topic", c.reader.Config().Topic))
				return nil
			}
			c.logger.Error("Error fetching message from Kafka", zap.Error(err), zap.String("topic", c.reader.Config().Topic))
			// Depending on the error, you might want to retry or exit.
			// For simplicity, we'll continue, but in production, you might need more robust error handling.
			time.Sleep(1 * time.Second) // Avoid busy-looping on persistent errors
			continue
		}

		handleCtx, cancelHandler := context.WithTimeout(context.Background(), 25*time.Second) // New context for handler
		if err := c.handler(handleCtx, m); err != nil {
			c.logger.Error("Error handling Kafka message",
				zap.String("topic", m.Topic),
				zap.Int("partition", m.Partition),
				zap.Int64("offset", m.Offset),
				zap.Error(err))
			// Decide if you want to commit or not based on the error.
			// For now, we continue without committing if handler fails.
			cancelHandler()
			continue
		}
		cancelHandler()

		commitCtx, cancelCommit := context.WithTimeout(context.Background(), 5*time.Second) // New context for commit
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
