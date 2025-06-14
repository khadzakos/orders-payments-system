package outbox_repo

import (
	"context"
	"time"
)

type OutboxStatus string

const (
	StatusPending OutboxStatus = "PENDING"
	StatusSent    OutboxStatus = "SENT"
	StatusFailed  OutboxStatus = "FAILED"
)

type OutboxMessage struct {
	ID        string       `json:"id"`
	Topic     string       `json:"topic"`
	Payload   []byte       `json:"payload"`
	Status    OutboxStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	SentAt    *time.Time   `json:"sent_at"`
}

type OutboxRepository interface {
	CreateMessage(ctx context.Context, msg *OutboxMessage) error
	GetUnsentMessages(ctx context.Context) ([]*OutboxMessage, error)
	MarkMessageSent(ctx context.Context, id string) error
}
