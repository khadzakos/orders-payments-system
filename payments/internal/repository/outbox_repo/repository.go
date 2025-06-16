package outbox_repo

import (
	"context"
	"payments/internal/domain"
)

type OutboxRepository interface {
	CreateMessageTx(ctx context.Context, querier domain.Querier, msg *domain.OutboxMessage) error
	GetPendingMessages(ctx context.Context, querier domain.Querier, limit int) ([]domain.OutboxMessage, error)
	UpdateMessageStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.OutboxMessageStatus) error // Changed to OutboxMessageStatus
}
