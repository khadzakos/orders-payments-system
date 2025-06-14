package payments_repo

import (
	"context"
	"database/sql"
	"payments/internal/domain"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	CreateTx(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error
	Update(ctx context.Context, payment *domain.Payment) error
	UpdateTx(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	GetByIDTx(ctx context.Context, tx *sql.Tx, id string) (*domain.Payment, error)
	GetByOrderIDTx(ctx context.Context, tx *sql.Tx, orderID string) (*domain.Payment, error)
}
