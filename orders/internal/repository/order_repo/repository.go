package order_repo

import (
	"context"
	"orders/internal/domain"
	"orders/internal/repository/outbox_repo"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) error
	CreateOrderAndOutboxMessage(ctx context.Context, order *domain.Order, msg *outbox_repo.OutboxMessage) error
	GetOrderByID(ctx context.Context, id string) (*domain.Order, error)
	GetOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error)
	GetAllOrders(ctx context.Context) ([]*domain.Order, error)
	UpdateOrder(ctx context.Context, order *domain.Order) error
}
