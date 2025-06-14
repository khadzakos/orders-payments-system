package domain

import "time"

type Account struct {
	ID        string
	UserID    string
	Balance   float64
	CreatedAt time.Time
	UpdatedAt time.Time
}
