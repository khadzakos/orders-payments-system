package orders

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"orders/internal/domain"
	"orders/internal/infrastructure/kafka"
	"orders/internal/repository/order_repo"
	"orders/internal/repository/outbox_repo"
	"orders/internal/util"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrInvalidOrder  = errors.New("invalid order data")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*OrderResponse, error)
	GetOrdersByUserID(ctx context.Context, userID string) ([]*OrderResponse, error)
	GetAllOrders(ctx context.Context) ([]*OrderResponse, error)
	HandlePaymentStatusUpdate(ctx context.Context, event *PaymentStatusEvent) error
	ProcessOutbox(ctx context.Context) error
}

type orderService struct {
	orderRepo     order_repo.OrderRepository
	outboxRepo    outbox_repo.OutboxRepository
	kafkaProducer kafka.Producer
	logger        *zap.Logger
}

func NewOrderService(
	orderRepo order_repo.OrderRepository,
	outboxRepo outbox_repo.OutboxRepository,
	kafkaProducer kafka.Producer,
	logger *zap.Logger,
) OrderService {
	return &orderService{
		orderRepo:     orderRepo,
		outboxRepo:    outboxRepo,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderResponse, error) {
	orderID := util.GenerateUUID()
	order, err := domain.NewOrder(orderID, req.UserID, req.Description, req.Amount)
	if err != nil {
		s.logger.Error("Failed to create new order domain object", zap.Error(err))
		return nil, ErrInvalidOrder
	}

	paymentTaskPayload := map[string]interface{}{
		"order_id":    order.ID,
		"user_id":     order.UserID,
		"amount":      order.Amount,
		"description": order.Description,
	}
	payloadBytes, err := json.Marshal(paymentTaskPayload)
	if err != nil {
		s.logger.Error("Failed to marshal payment task payload", zap.String("order_id", order.ID), zap.Error(err))
		return nil, errors.New("internal server error")
	}

	outboxMessage := &outbox_repo.OutboxMessage{
		ID:        util.GenerateUUID(),
		Topic:     "order_payment_tasks",
		Payload:   payloadBytes,
		Status:    outbox_repo.StatusPending,
		CreatedAt: time.Now(),
	}

	err = s.orderRepo.CreateOrderAndOutboxMessage(ctx, order, outboxMessage)
	if err != nil {
		s.logger.Error("Failed to save order and outbox message for order", zap.String("order_id", order.ID), zap.Error(err))
		return nil, errors.New("failed to initiate payment process")
	}

	s.logger.Info("Order created and payment task added to outbox", zap.String("order_id", order.ID))

	return mapOrderToResponse(order), nil
}

func (s *orderService) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("Order not found", zap.String("order_id", orderID))
			return nil, ErrOrderNotFound
		}
		s.logger.Error("Failed to get order from repository", zap.String("order_id", orderID), zap.Error(err))
		return nil, errors.New("internal server error")
	}
	return mapOrderToResponse(order), nil
}

func (s *orderService) GetOrdersByUserID(ctx context.Context, userID string) ([]*OrderResponse, error) {
	orders, err := s.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get orders for user from repository", zap.String("user_id", userID), zap.Error(err))
		return nil, errors.New("internal server error")
	}
	return mapOrdersToResponse(orders), nil
}

func (s *orderService) GetAllOrders(ctx context.Context) ([]*OrderResponse, error) {
	orders, err := s.orderRepo.GetAllOrders(ctx)
	if err != nil {
		s.logger.Error("Failed to get all orders from repository", zap.Error(err))
		return nil, errors.New("internal server error")
	}
	return mapOrdersToResponse(orders), nil
}

func (s *orderService) HandlePaymentStatusUpdate(ctx context.Context, event *PaymentStatusEvent) error {
	order, err := s.orderRepo.GetOrderByID(ctx, event.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Warn("Order not found for payment status update, ignoring",
				zap.String("order_id", event.OrderID),
				zap.String("payment_status", event.Status))
			return nil
		}
		s.logger.Error("Failed to retrieve order for payment status update", zap.String("order_id", event.OrderID), zap.Error(err))
		return errors.New("failed to retrieve order for update")
	}

	originalStatus := order.Status

	var updateErr error
	switch event.Status {
	case "SUCCESS":
		updateErr = order.MarkAsFinished()
	case "FAILED":
		updateErr = order.MarkAsCancelled()
	default:
		s.logger.Warn("Received unknown payment status",
			zap.String("order_id", event.OrderID),
			zap.String("received_status", event.Status))
		return fmt.Errorf("unknown payment status: %s", event.Status)
	}

	if updateErr != nil {
		s.logger.Warn("Failed to apply status change based on payment event",
			zap.String("order_id", order.ID),
			zap.String("current_status", string(originalStatus)),
			zap.String("event_status", event.Status),
			zap.Error(updateErr))
		return nil
	}

	if originalStatus == order.Status {
		s.logger.Info("Order status already matches payment event status, no update needed",
			zap.String("order_id", order.ID),
			zap.String("status", string(originalStatus)))
		return nil
	}

	err = s.orderRepo.UpdateOrder(ctx, order)
	if err != nil {
		s.logger.Error("Failed to update order status in database",
			zap.String("order_id", order.ID),
			zap.String("old_status", string(originalStatus)),
			zap.String("new_status", string(order.Status)),
			zap.Error(err))
		return errors.New("failed to update order status")
	}

	s.logger.Info("Order status updated based on payment event",
		zap.String("order_id", order.ID),
		zap.String("old_status", string(originalStatus)),
		zap.String("new_status", string(order.Status)))
	return nil
}

func (s *orderService) ProcessOutbox(ctx context.Context) error {
	messages, err := s.outboxRepo.GetUnsentMessages(ctx)
	if err != nil {
		s.logger.Error("Failed to get unsent outbox messages", zap.Error(err))
		return fmt.Errorf("failed to get unsent outbox messages: %w", err)
	}

	if len(messages) == 0 {
		s.logger.Debug("No unsent outbox messages found.")
		return nil
	}

	s.logger.Info("Processing unsent outbox messages", zap.Int("count", len(messages)))

	for _, msg := range messages {
		if err := s.kafkaProducer.Produce(ctx, msg.Topic, msg.Payload); err != nil {
			s.logger.Error("Failed to produce outbox message to Kafka",
				zap.String("message_id", msg.ID),
				zap.String("topic", msg.Topic),
				zap.Error(err))
			continue
		}
		if err := s.outboxRepo.MarkMessageSent(ctx, msg.ID); err != nil {
			s.logger.Error("Failed to mark outbox message as sent",
				zap.String("message_id", msg.ID),
				zap.Error(err))
		} else {
			s.logger.Debug("Outbox message sent and marked as sent", zap.String("message_id", msg.ID))
		}
	}
	return nil
}

func mapOrderToResponse(order *domain.Order) *OrderResponse {
	return &OrderResponse{
		ID:          order.ID,
		UserID:      order.UserID,
		Amount:      order.Amount,
		Description: order.Description,
		Status:      string(order.Status),
	}
}

func mapOrdersToResponse(orders []*domain.Order) []*OrderResponse {
	responses := make([]*OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = mapOrderToResponse(order)
	}
	return responses
}
