package outbox_repo

import (
	"context"
	"database/sql"
	"payments/internal/domain"
)

type OutboxRepository interface {
	CreateMessageTx(ctx context.Context, tx *sql.Tx, msg *domain.OutboxMessage) error
	GetPendingMessages(ctx context.Context) ([]domain.OutboxMessage, error)
	MarkMessagesAsSent(ctx context.Context, ids []string) error
	MarkMessagesAsFailed(ctx context.Context, ids []string) error
}
