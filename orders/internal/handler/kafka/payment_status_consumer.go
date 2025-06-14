package kafka

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"orders/internal/app/orders"
)

type PaymentStatusConsumer struct {
	orderService orders.OrderService
	logger       *zap.Logger
}

func NewPaymentStatusConsumer(s orders.OrderService, l *zap.Logger) *PaymentStatusConsumer {
	return &PaymentStatusConsumer{orderService: s, logger: l}
}

func (c *PaymentStatusConsumer) HandleMessage(ctx context.Context, message []byte) error {
	var event orders.PaymentStatusEvent
	if err := json.Unmarshal(message, &event); err != nil {
		c.logger.Error("Error unmarshalling Kafka message", zap.Error(err), zap.String("raw_message", string(message)))
		return nil
	}

	c.logger.Info("Received payment status update",
		zap.String("order_id", event.OrderID),
		zap.String("status", event.Status))

	err := c.orderService.HandlePaymentStatusUpdate(ctx, &event)
	if err != nil {
		c.logger.Error("Error processing payment status update for order",
			zap.String("order_id", event.OrderID),
			zap.Error(err))
		return err
	}
	return nil
}
