package domain

import "time"

type InboxMessageStatus string

const (
	InboxStatusNew        InboxMessageStatus = "NEW"
	InboxStatusProcessing InboxMessageStatus = "PROCESSING"
	InboxStatusProcessed  InboxMessageStatus = "PROCESSED"
	InboxStatusFailed     InboxMessageStatus = "FAILED"
)

// InboxMessage представляет запись в таблице Inbox для обеспечения exactly-once семантики.
type InboxMessage struct {
	ID             string
	KafkaTopic     string
	KafkaPartition int
	KafkaOffset    int64
	ConsumerGroup  string
	Payload        []byte
	Status         InboxMessageStatus
	ReceivedAt     time.Time
	ProcessedAt    *time.Time
}
