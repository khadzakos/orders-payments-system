package domain

import (
	"errors"
	"time"
)

type OrderStatus string

const (
	OrderStatusNew            OrderStatus = "NEW"
	OrderStatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	OrderStatusFinished       OrderStatus = "FINISHED"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
)

type Order struct {
	ID          string
	UserID      string
	Amount      float64
	Description string
	Status      OrderStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewOrder(id, userID, description string, amount float64) (*Order, error) {
	if id == "" || userID == "" || amount <= 0 {
		return nil, errors.New("invalid order data")
	}
	now := time.Now()
	return &Order{
		ID:          id,
		UserID:      userID,
		Amount:      amount,
		Description: description,
		Status:      OrderStatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (o *Order) MarkAsFinished() error {
	if o.Status == OrderStatusCancelled {
		return errors.New("cannot finish a cancelled order")
	}
	o.Status = OrderStatusFinished
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) MarkAsCancelled() error {
	if o.Status == OrderStatusFinished {
		return errors.New("cannot cancel a finished order")
	}
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()
	return nil
}

func (o *Order) MarkAsPendingPayment() error {
	if o.Status != OrderStatusNew {
		return errors.New("order must be in NEW status to become PENDING_PAYMENT")
	}
	o.Status = OrderStatusPendingPayment
	o.UpdatedAt = time.Now()
	return nil
}
