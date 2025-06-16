package payments_repo

import (
	"context"
	"payments/internal/domain"
)

type PaymentRepository interface {
	CreateTx(ctx context.Context, querier domain.Querier, payment *domain.Payment) error
	GetByIDTx(ctx context.Context, querier domain.Querier, id string) (*domain.Payment, error)
	GetByOrderIDTx(ctx context.Context, querier domain.Querier, orderID string) (*domain.Payment, error)
	UpdateStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.PaymentStatus) error
}
