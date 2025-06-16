package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusNew       PaymentStatus = "NEW"
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusCompleted PaymentStatus = "COMPLETED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
)

type Payment struct {
	ID        string
	OrderID   string
	UserID    int64
	Amount    float64
	Status    PaymentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
