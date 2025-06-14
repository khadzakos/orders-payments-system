package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/domain"
	"payments/internal/repository/inbox_repo"
)

type InboxRepository struct {
	db *sql.DB
}

func NewInboxRepository(db *sql.DB) *InboxRepository {
	return &InboxRepository{db: db}
}

func (r *InboxRepository) CreateMessageTx(ctx context.Context, tx *sql.Tx, msg *domain.InboxMessage) error {
	query := `
		INSERT INTO inbox_messages (id, kafka_topic, kafka_partition, kafka_offset, consumer_group, payload, status, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (kafka_topic, kafka_partition, kafka_offset, consumer_group) DO NOTHING
		RETURNING id, status -- Получаем id и статус, если запись уже существовала
	`

	var existingID string
	var existingStatus string
	err := tx.QueryRowContext(ctx, query,
		msg.ID,
		msg.KafkaTopic,
		msg.KafkaPartition,
		msg.KafkaOffset,
		msg.ConsumerGroup,
		msg.Payload,
		msg.Status,
		msg.ReceivedAt,
	).Scan(&existingID, &existingStatus)

	if err != nil {
		if err == sql.ErrNoRows {
			existingMsg, getErr := r.GetMessageByKafkaMetadataTx(ctx, tx,
				msg.KafkaTopic, msg.KafkaPartition, msg.KafkaOffset, msg.ConsumerGroup)
			if getErr != nil {
				return fmt.Errorf("failed to retrieve existing inbox message after conflict: %w", getErr)
			}
			if existingMsg.Status == domain.InboxStatusProcessed {
				return inbox_repo.ErrMessageAlreadyProcessed
			}
			return inbox_repo.ErrMessageAlreadyPending
		}
		return fmt.Errorf("failed to insert inbox message: %w", err)
	}

	if existingID == msg.ID {
		return nil
	}
	if existingStatus == string(domain.InboxStatusProcessed) {
		return inbox_repo.ErrMessageAlreadyProcessed
	}
	return inbox_repo.ErrMessageAlreadyPending
}

func (r *InboxRepository) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id string, status domain.InboxMessageStatus) error {
	query := `
		UPDATE inbox_messages
		SET status = $1, processed_at = CASE WHEN $1 = 'PROCESSED' THEN $2 ELSE NULL END
		WHERE id = $3
	`
	res, err := tx.ExecContext(ctx, query, string(status), time.Now(), id)
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

func (r *InboxRepository) GetMessageByKafkaMetadataTx(ctx context.Context, tx *sql.Tx, topic string, partition int, offset int64, consumerGroup string) (*domain.InboxMessage, error) {
	query := `
		SELECT id, kafka_topic, kafka_partition, kafka_offset, consumer_group, payload, status, received_at, processed_at
		FROM inbox_messages
		WHERE kafka_topic = $1 AND kafka_partition = $2 AND kafka_offset = $3 AND consumer_group = $4
	`
	msg := &domain.InboxMessage{}
	var processedAt sql.NullTime
	err := tx.QueryRowContext(ctx, query, topic, partition, offset, consumerGroup).Scan(
		&msg.ID,
		&msg.KafkaTopic,
		&msg.KafkaPartition,
		&msg.KafkaOffset,
		&msg.ConsumerGroup,
		&msg.Payload,
		&msg.Status,
		&msg.ReceivedAt,
		&processedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get inbox message by kafka metadata: %w", err)
	}
	if processedAt.Valid {
		msg.ProcessedAt = &processedAt.Time
	}
	return msg, nil
}
