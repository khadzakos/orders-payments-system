package domain

import (
	"fmt"
	"time"
)

type InboxMessageStatus string

const (
	InboxStatusNew        InboxMessageStatus = "NEW"
	InboxStatusProcessing InboxMessageStatus = "PROCESSING"
	InboxStatusCompleted  InboxMessageStatus = "COMPLETED"
	InboxStatusFailed     InboxMessageStatus = "FAILED"
)

var (
	ErrMessageAlreadyProcessed = fmt.Errorf("inbox message already processed")
	ErrMessageAlreadyPending   = fmt.Errorf("inbox message already pending")
)

// InboxMessage представляет запись в таблице Inbox.
type InboxMessage struct {
	ID          string
	OrderID     string
	Payload     []byte
	Status      InboxMessageStatus
	ReceivedAt  time.Time
	ProcessedAt *time.Time
}
