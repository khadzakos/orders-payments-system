package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"orders/internal/repository/outbox_repo"
)

type pgOutboxRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewOutboxRepository(db *sql.DB, l *zap.Logger) outbox_repo.OutboxRepository {
	return &pgOutboxRepository{db: db, logger: l}
}

func (r *pgOutboxRepository) CreateMessage(ctx context.Context, msg *outbox_repo.OutboxMessage) error {
	query := `INSERT INTO outbox_messages (id, topic, payload, status, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.Topic, msg.Payload, msg.Status, msg.CreatedAt)
	if err != nil {
		r.logger.Error("Failed to create outbox message", zap.String("message_id", msg.ID), zap.Error(err))
		return fmt.Errorf("failed to create outbox message: %w", err)
	}
	r.logger.Debug("Outbox message created", zap.String("message_id", msg.ID), zap.String("topic", msg.Topic))
	return nil
}

func (r *pgOutboxRepository) GetUnsentMessages(ctx context.Context) ([]*outbox_repo.OutboxMessage, error) {
	var messages []*outbox_repo.OutboxMessage
	query := `SELECT id, topic, payload, status, created_at, sent_at FROM outbox_messages WHERE status = $1 ORDER BY created_at ASC FOR UPDATE SKIP LOCKED LIMIT 100`
	rows, err := r.db.QueryContext(ctx, query, outbox_repo.StatusPending)
	if err != nil {
		r.logger.Error("Failed to get unsent outbox messages", zap.Error(err))
		return nil, fmt.Errorf("failed to get unsent outbox messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		msg := &outbox_repo.OutboxMessage{}
		var sentAt sql.NullTime
		if err := rows.Scan(&msg.ID, &msg.Topic, &msg.Payload, &msg.Status, &msg.CreatedAt, &sentAt); err != nil {
			r.logger.Error("Failed to scan outbox message row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan outbox message row: %w", err)
		}
		if sentAt.Valid {
			msg.SentAt = &sentAt.Time
		}
		messages = append(messages, msg)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Rows error while getting unsent outbox messages", zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return messages, nil
}

func (r *pgOutboxRepository) MarkMessageSent(ctx context.Context, id string) error {
	query := `UPDATE outbox_messages SET status = $1, sent_at = $2 WHERE id = $3 AND status = $4`
	res, err := r.db.ExecContext(ctx, query, outbox_repo.StatusSent, time.Now(), id, outbox_repo.StatusPending)
	if err != nil {
		r.logger.Error("Failed to mark outbox message as sent", zap.String("message_id", id), zap.Error(err))
		return fmt.Errorf("failed to mark outbox message %s as sent: %w", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected for outbox message mark as sent", zap.String("message_id", id), zap.Error(err))
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rowsAffected == 0 {
		r.logger.Warn("No rows affected when marking outbox message as sent, it might be already sent or not exist", zap.String("message_id", id))
	} else {
		r.logger.Debug("Outbox message marked as sent", zap.String("message_id", id))
	}
	return nil
}
