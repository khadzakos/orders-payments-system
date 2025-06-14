package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/domain"

	"github.com/lib/pq"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) CreateMessageTx(ctx context.Context, tx *sql.Tx, msg *domain.OutboxMessage) error {
	query := `
		INSERT INTO outbox_messages (id, aggregate_id, aggregate_type, message_type, topic, key_value, payload, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.ExecContext(ctx, query,
		msg.ID,
		msg.AggregateID,
		msg.AggregateType,
		msg.MessageType,
		msg.Topic,
		msg.Key,
		msg.Payload,
		msg.Status,
		msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox message: %w", err)
	}
	return nil
}

func (r *OutboxRepository) GetPendingMessages(ctx context.Context) ([]domain.OutboxMessage, error) {
	query := `
		SELECT id, aggregate_id, aggregate_type, message_type, topic, key_value, payload, status, created_at, sent_at
		FROM outbox_messages
		WHERE status = $1
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.db.QueryContext(ctx, query, domain.OutboxStatusPending)
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
			&msg.AggregateID,
			&msg.AggregateType,
			&msg.MessageType,
			&msg.Topic,
			&msg.Key,
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

func (r *OutboxRepository) MarkMessagesAsSent(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := `
		UPDATE outbox_messages
		SET status = $1, sent_at = $2
		WHERE id = ANY($3)
	`
	res, err := r.db.ExecContext(ctx, query, domain.OutboxStatusSent, time.Now(), pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to mark outbox messages as sent: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for outbox sent: %w", err)
	}
	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("not all outbox messages were marked as sent; expected %d, got %d", len(ids), rowsAffected)
	}
	return nil
}

func (r *OutboxRepository) MarkMessagesAsFailed(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := `
		UPDATE outbox_messages
		SET status = $1, sent_at = NULL
		WHERE id = ANY($2)
	`
	res, err := r.db.ExecContext(ctx, query, domain.OutboxStatusFailed, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to mark outbox messages as failed: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for outbox failed: %w", err)
	}
	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("not all outbox messages were marked as failed; expected %d, got %d", len(ids), rowsAffected)
	}
	return nil
}
