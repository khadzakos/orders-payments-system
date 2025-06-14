package domain

import "time"

type OutboxMessageStatus string

const (
	OutboxStatusPending OutboxMessageStatus = "PENDING"
	OutboxStatusSent    OutboxMessageStatus = "SENT"
	OutboxStatusFailed  OutboxMessageStatus = "FAILED"
)

// OutboxMessage представляет сообщение, ожидающее отправки в Kafka, с бизнес-контекстом.
type OutboxMessage struct {
	ID            string
	AggregateID   string
	AggregateType string
	MessageType   string
	Topic         string
	Key           string
	Payload       []byte
	Status        OutboxMessageStatus
	CreatedAt     time.Time
	SentAt        *time.Time
}
