package inbox_repo

import (
	"context"
	"payments/internal/domain"
)

type InboxRepository interface {
	CreateMessageTx(ctx context.Context, querier domain.Querier, msg *domain.InboxMessage) error
	UpdateStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.InboxMessageStatus) error
	GetMessageByOrderIDTx(ctx context.Context, querier domain.Querier, orderID string) (*domain.InboxMessage, error)
}
