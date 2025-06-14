package accounts_repo

import (
	"context"
	"database/sql"
	"fmt"

	"payments/internal/domain"
)

var (
	ErrInsufficientFunds = fmt.Errorf("insufficient funds")
	ErrAccountNotFound   = fmt.Errorf("account not found")
)

type AccountRepository interface {
	GetAccountForUserTx(ctx context.Context, tx *sql.Tx, userID string) (*domain.Account, error)
	UpdateBalanceTx(ctx context.Context, tx *sql.Tx, accountID string, amount float64) error
	CreateAccountTx(ctx context.Context, tx *sql.Tx, account *domain.Account) error
}
