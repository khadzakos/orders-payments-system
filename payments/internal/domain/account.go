package domain

import (
	"errors"
	"time"
)

var ErrAccountNotFound = errors.New("account not found")
var ErrAccountAlreadyExists = errors.New("account already exists")
var ErrInsufficientFunds = errors.New("insufficient funds")

type Account struct {
	ID        string
	UserID    int64
	Balance   float64
	CreatedAt time.Time
	UpdatedAt time.Time
}
