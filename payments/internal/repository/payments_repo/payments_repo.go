package payments_repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/domain"
)

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *paymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateTx(ctx context.Context, querier domain.Querier, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, user_id, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := querier.ExecContext(ctx, query,
		payment.ID,
		payment.OrderID,
		payment.UserID,
		payment.Amount,
		payment.Status,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

func (r *paymentRepository) GetByIDTx(ctx context.Context, querier domain.Querier, id string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount, status, created_at, updated_at
		FROM payments
		WHERE id = $1
	`
	payment := &domain.Payment{}
	err := querier.QueryRowContext(ctx, query, id).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.UserID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("payment with id %s not found: %w", id, err)
		}
		return nil, fmt.Errorf("failed to get payment by id %s: %w", id, err)
	}
	return payment, nil
}

func (r *paymentRepository) GetByOrderIDTx(ctx context.Context, querier domain.Querier, orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount, status, created_at, updated_at
		FROM payments
		WHERE order_id = $1
	`
	payment := &domain.Payment{}
	err := querier.QueryRowContext(ctx, query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.UserID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get payment by order id %s: %w", orderID, err)
	}
	return payment, nil
}

func (r *paymentRepository) UpdateStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.PaymentStatus) error {
	query := `
		UPDATE payments
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	res, err := querier.ExecContext(ctx, query, string(status), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update payment status %s: %w", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for payment status update: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("payment with id %s not found for status update", id)
	}
	return nil
}
