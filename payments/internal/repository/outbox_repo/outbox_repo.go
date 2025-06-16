package outbox_repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/domain"
)

type outboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *outboxRepository {
	return &outboxRepository{db: db}
}

func (r *outboxRepository) CreateMessageTx(ctx context.Context, querier domain.Querier, msg *domain.OutboxMessage) error {
	query := `
		INSERT INTO outbox_messages (id, order_id, order_status, payload, status, created_at, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var sentAt sql.NullTime
	if msg.SentAt != nil {
		sentAt = sql.NullTime{Time: *msg.SentAt, Valid: true}
	}

	_, err := querier.ExecContext(ctx, query,
		msg.ID,
		msg.OrderID,
		msg.OrderStatus,
		msg.Payload,
		msg.Status,
		msg.CreatedAt,
		sentAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}
	return nil
}

func (r *outboxRepository) GetPendingMessages(ctx context.Context, querier domain.Querier, limit int) ([]domain.OutboxMessage, error) {
	query := `
		SELECT id, order_id, order_status, payload, status, created_at, sent_at
		FROM outbox_messages
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`
	rows, err := querier.QueryContext(ctx, query, domain.OutboxStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending outbox messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.OutboxMessage
	for rows.Next() {
		msg := domain.OutboxMessage{}
		var sentAt sql.NullTime
		err := rows.Scan(
			&msg.ID,
			&msg.OrderID,
			&msg.OrderStatus,
			&msg.Payload,
			&msg.Status,
			&msg.CreatedAt,
			&sentAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox message: %w", err)
		}
		if sentAt.Valid {
			msg.SentAt = &sentAt.Time
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outbox messages: %w", err)
	}

	return messages, nil
}

func (r *outboxRepository) UpdateMessageStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.OutboxMessageStatus) error {
	query := `
		UPDATE outbox_messages
		SET status = $1, sent_at = $2
		WHERE id = $3
	`
	var sentAt sql.NullTime
	if status == domain.OutboxStatusSent {
		sentAt = sql.NullTime{Time: time.Now(), Valid: true}
	} else {
		// For FAILED or other statuses, clear sent_at
		sentAt = sql.NullTime{Valid: false}
	}

	res, err := querier.ExecContext(ctx, query, status, sentAt, id)
	if err != nil {
		return fmt.Errorf("failed to update outbox message status for id %s: %w", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for outbox update (id %s): %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no outbox message found with id %s to update status", id)
	}
	return nil
}
