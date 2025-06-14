package inbox_repo

import (
	"context"
	"database/sql"
	"fmt"
	"payments/internal/domain"
)

type InboxRepository interface {
	CreateMessageTx(ctx context.Context, tx *sql.Tx, msg *domain.InboxMessage) error
	UpdateStatusTx(ctx context.Context, tx *sql.Tx, id string, status domain.InboxMessageStatus) error
	GetMessageByKafkaMetadataTx(ctx context.Context, tx *sql.Tx, topic string, partition int, offset int64, consumerGroup string) (*domain.InboxMessage, error)
}

var (
	ErrMessageAlreadyProcessed = fmt.Errorf("inbox message already processed")
	ErrMessageAlreadyPending   = fmt.Errorf("inbox message already pending")
)
