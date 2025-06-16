package inbox_repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/domain"

	"github.com/lib/pq"
)

type inboxRepository struct {
	db *sql.DB
}

func NewInboxRepository(db *sql.DB) *inboxRepository {
	return &inboxRepository{db: db}
}

func (r *inboxRepository) CreateMessageTx(ctx context.Context, querier domain.Querier, msg *domain.InboxMessage) error {
	query := `
		INSERT INTO inbox_messages (id, order_id, payload, status, received_at, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	var processedAt sql.NullTime
	if msg.ProcessedAt != nil {
		processedAt = sql.NullTime{Time: *msg.ProcessedAt, Valid: true}
	}

	_, err := querier.ExecContext(ctx, query,
		msg.ID,
		msg.OrderID,
		msg.Payload,
		msg.Status,
		msg.ReceivedAt,
		processedAt,
	)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return fmt.Errorf("inbox message with id %s already exists: %w", msg.ID, err)
		}
		return fmt.Errorf("failed to create inbox message: %w", err)
	}
	return nil
}

func (r *inboxRepository) UpdateStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.InboxMessageStatus) error {
	query := `
		UPDATE inbox_messages
		SET status = $1, processed_at = CASE WHEN $1::VARCHAR = 'COMPLETED' THEN $2 ELSE processed_at END
		WHERE id = $3
	`
	res, err := querier.ExecContext(ctx, query, string(status), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update inbox message status %s: %w", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for inbox message update: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("inbox message with id %s not found for status update", id)
	}
	return nil
}

func (r *inboxRepository) GetMessageByOrderIDTx(ctx context.Context, querier domain.Querier, orderID string) (*domain.InboxMessage, error) {
	query := `
		SELECT id, order_id, payload, status, received_at, processed_at
		FROM inbox_messages
		WHERE order_id = $1
	`
	msg := &domain.InboxMessage{}
	var processedAt sql.NullTime
	err := querier.QueryRowContext(ctx, query, orderID).Scan(
		&msg.ID,
		&msg.OrderID,
		&msg.Payload,
		&msg.Status,
		&msg.ReceivedAt,
		&processedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get inbox message by order_id %s: %w", orderID, err)
	}
	if processedAt.Valid {
		msg.ProcessedAt = &processedAt.Time
	}
	return msg, nil
}
