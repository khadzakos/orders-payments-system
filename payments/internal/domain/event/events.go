package event

import "time"

type OrderCreatedEvent struct {
	OrderID   string    `json:"order_id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

type PaymentProcessedEvent struct {
	PaymentID     string    `json:"payment_id"`
	OrderID       string    `json:"order_id"`
	UserID        int64     `json:"user_id"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	TransactionID string    `json:"transaction_id"`
	Timestamp     time.Time `json:"timestamp"`
}

type PaymentFailedEvent struct {
	PaymentID string    `json:"payment_id"`
	OrderID   string    `json:"order_id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}
