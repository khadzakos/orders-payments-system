package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"payments/internal/app/payments"
	"payments/internal/domain/event"
	kafka_infra "payments/internal/infrastucture/kafka"
)

func OrderCreatedMessageHandler(paymentService payments.PaymentService, logger *zap.Logger) kafka_infra.MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		logger.Info("Received Kafka message for order processing",
			zap.String("topic", msg.Topic),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
			zap.String("key", string(msg.Key)),
		)

		var orderCreatedEvent event.OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &orderCreatedEvent); err != nil {
			logger.Error("Failed to unmarshal Kafka message value to OrderCreatedEvent",
				zap.Error(err),
				zap.ByteString("value", msg.Value),
				zap.String("topic", msg.Topic),
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
			)
			return nil
		}

		logger.Info("Processing OrderCreatedEvent",
			zap.String("order_id", orderCreatedEvent.OrderID),
			zap.Int64("user_id", orderCreatedEvent.UserID),
			zap.Float64("amount", orderCreatedEvent.Amount),
		)

		processErr := paymentService.ProcessIncomingOrderCreatedEvent(
			ctx,
			orderCreatedEvent.OrderID,
			orderCreatedEvent.OrderID,
			orderCreatedEvent.UserID,
			orderCreatedEvent.Amount,
			msg.Value,
		)

		if processErr != nil {
			logger.Error("Failed to process order created event",
				zap.String("order_id", orderCreatedEvent.OrderID),
				zap.Error(processErr),
			)
			return fmt.Errorf("failed to process order created event for order %s: %w", orderCreatedEvent.OrderID, processErr)
		}

		logger.Info("Successfully processed order created event",
			zap.String("order_id", orderCreatedEvent.OrderID),
		)
		return nil
	}
}
