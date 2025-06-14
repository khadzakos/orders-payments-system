package kafka_infra

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// MessageHandler определяет контракт для обработчика Kafka-сообщений.
type MessageHandler func(ctx context.Context, msg kafka.Message) error

// Consumer определяет интерфейс для Kafka-потребителя.
type Consumer interface {
	Start(ctx context.Context, handler MessageHandler) error
	Stop()
}

type kafkaConsumer struct {
	reader  *kafka.Reader
	logger  *zap.Logger
	brokers []string
	topic   string
	groupID string
	cancel  context.CancelFunc
	started chan struct{}
}

func NewConsumer(brokerURLs []string, groupID, topic string, logger *zap.Logger) Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:                brokerURLs,
		GroupID:                groupID,
		Topic:                  topic,
		MinBytes:               10e3,
		MaxBytes:               10e6,
		ReadBatchTimeout:       1 * time.Second,
		Logger:                 kafka.LoggerFunc(func(msg string, args ...interface{}) { logger.Debug(fmt.Sprintf(msg, args...)) }),
		ErrorLogger:            kafka.LoggerFunc(func(msg string, args ...interface{}) { logger.Error(fmt.Sprintf(msg, args...)) }),
		HeartbeatInterval:      3 * time.Second,
		CommitInterval:         time.Second,
		PartitionWatchInterval: 5 * time.Second,
		MaxAttempts:            3,
	})

	return &kafkaConsumer{
		reader:  reader,
		logger:  logger,
		brokers: brokerURLs,
		topic:   topic,
		groupID: groupID,
		started: make(chan struct{}),
	}
}

func (c *kafkaConsumer) Start(ctx context.Context, handler MessageHandler) error {
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.logger.Info("Kafka consumer starting", zap.String("topic", c.topic), zap.String("group_id", c.groupID))

	close(c.started)

	for {
		select {
		case <-consumerCtx.Done():
			c.logger.Info("Kafka consumer context cancelled, stopping reader.")
			return c.reader.Close()
		default:
			msg, err := c.reader.FetchMessage(consumerCtx)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					c.logger.Info("Kafka FetchMessage context cancelled or timed out.")
					continue
				}
				c.logger.Error("Failed to fetch message from Kafka", zap.Error(err))
				time.Sleep(1 * time.Second)
				continue
			}

			c.logger.Info("Received Kafka message",
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.String("key", string(msg.Key)),
			)

			if handlerErr := handler(consumerCtx, msg); handlerErr != nil {
				c.logger.Error("Error handling Kafka message, will not commit offset",
					zap.String("topic", msg.Topic),
					zap.Int("partition", msg.Partition),
					zap.Int64("offset", msg.Offset),
					zap.Error(handlerErr),
				)

			} else {
				if commitErr := c.reader.CommitMessages(consumerCtx, msg); commitErr != nil {
					c.logger.Error("Failed to commit offset for Kafka message",
						zap.String("topic", msg.Topic),
						zap.Int("partition", msg.Partition),
						zap.Int64("offset", msg.Offset),
						zap.Error(commitErr),
					)
				} else {
					c.logger.Debug("Kafka message offset committed",
						zap.String("topic", msg.Topic),
						zap.Int("partition", msg.Partition),
						zap.Int64("offset", msg.Offset),
					)
				}
			}
		}
	}
}

func (c *kafkaConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.logger.Info("Kafka consumer stop signal sent.")
}
