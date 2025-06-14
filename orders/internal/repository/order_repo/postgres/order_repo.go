package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"orders/internal/domain"
	"orders/internal/repository/order_repo"
	"orders/internal/repository/outbox_repo"
)

type pgOrderRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewOrderRepository(db *sql.DB, l *zap.Logger) order_repo.OrderRepository {
	return &pgOrderRepository{db: db, logger: l}
}

func (r *pgOrderRepository) CreateOrderAndOutboxMessage(ctx context.Context, order *domain.Order, msg *outbox_repo.OutboxMessage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction for order and outbox message creation", zap.String("order_id", order.ID), zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			r.logger.Error("Panic during transaction for order and outbox message creation, rolling back", zap.String("order_id", order.ID))
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			r.logger.Warn("Rolling back transaction for order and outbox message creation due to error", zap.String("order_id", order.ID), zap.Error(err))
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				r.logger.Error("Failed to commit transaction for order and outbox message creation", zap.String("order_id", order.ID), zap.Error(err))
			} else {
				r.logger.Debug("Transaction committed successfully for order and outbox message creation", zap.String("order_id", order.ID))
			}
		}
	}()

	orderQuery := `INSERT INTO orders (id, user_id, amount, description, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.ExecContext(ctx, orderQuery, order.ID, order.UserID, order.Amount, order.Description, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("tx failed to create order: %w", err)
	}
	r.logger.Debug("Order inserted in transaction", zap.String("order_id", order.ID))

	outboxQuery := `INSERT INTO outbox_messages (id, topic, payload, status, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.ExecContext(ctx, outboxQuery, msg.ID, msg.Topic, msg.Payload, msg.Status, msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("tx failed to create outbox message: %w", err)
	}
	r.logger.Debug("Outbox message inserted in transaction", zap.String("message_id", msg.ID))

	return err
}

func (r *pgOrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	query := `INSERT INTO orders (id, user_id, amount, description, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, order.ID, order.UserID, order.Amount, order.Description, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		r.logger.Error("Failed to create order", zap.String("order_id", order.ID), zap.Error(err))
		return fmt.Errorf("failed to create order: %w", err)
	}
	r.logger.Debug("Order created successfully", zap.String("order_id", order.ID))
	return nil
}

func (r *pgOrderRepository) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	order := &domain.Order{}
	query := `SELECT id, user_id, amount, description, status, created_at, updated_at FROM orders WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&order.ID, &order.UserID, &order.Amount, &order.Description, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		r.logger.Error("Failed to get order by ID", zap.String("order_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get order by ID %s: %w", id, err)
	}
	return order, nil
}

func (r *pgOrderRepository) GetOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	var orders []*domain.Order
	query := `SELECT id, user_id, amount, description, status, created_at, updated_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to query orders for user", zap.String("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("failed to get orders by user ID %s: %w", userID, err)
	}
	defer rows.Close()

	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(&order.ID, &order.UserID, &order.Amount, &order.Description, &order.Status, &order.CreatedAt, &order.UpdatedAt); err != nil {
			r.logger.Error("Failed to scan row for user orders", zap.String("user_id", userID), zap.Error(err))
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Rows error for user orders", zap.String("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return orders, nil
}

func (r *pgOrderRepository) GetAllOrders(ctx context.Context) ([]*domain.Order, error) {
	var orders []*domain.Order
	query := `SELECT id, user_id, amount, description, status, created_at, updated_at FROM orders ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to query all orders", zap.Error(err))
		return nil, fmt.Errorf("failed to get all orders: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(&order.ID, &order.UserID, &order.Amount, &order.Description, &order.Status, &order.CreatedAt, &order.UpdatedAt); err != nil {
			r.logger.Error("Failed to scan row for all orders", zap.Error(err))
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Rows error for all orders", zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return orders, nil
}

func (r *pgOrderRepository) UpdateOrder(ctx context.Context, order *domain.Order) error {
	query := `UPDATE orders SET user_id = $2, amount = $3, description = $4, status = $5, updated_at = $6 WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, order.ID, order.UserID, order.Amount, order.Description, order.Status, order.UpdatedAt)
	if err != nil {
		r.logger.Error("Failed to update order", zap.String("order_id", order.ID), zap.Error(err))
		return fmt.Errorf("failed to update order %s: %w", order.ID, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected for order update", zap.String("order_id", order.ID), zap.Error(err))
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rowsAffected == 0 {
		r.logger.Warn("No rows affected when updating order, order might not exist", zap.String("order_id", order.ID))
		return sql.ErrNoRows
	}
	r.logger.Debug("Order updated successfully", zap.String("order_id", order.ID), zap.String("new_status", string(order.Status)))
	return nil
}
