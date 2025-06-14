package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusNew       PaymentStatus = "NEW"
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusProcessed PaymentStatus = "PROCESSED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
)

type Payment struct {
	ID            string
	OrderID       string
	UserID        string
	Amount        float64
	Currency      string
	Status        PaymentStatus
	TransactionID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
