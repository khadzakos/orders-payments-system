package accounts_repo

import (
	"context"
	"payments/internal/domain"
)

type AccountRepository interface {
	CreateAccountTx(ctx context.Context, querier domain.Querier, account *domain.Account) error
	GetAccountForUserTx(ctx context.Context, querier domain.Querier, userID int64) (*domain.Account, error)
	UpdateBalanceTx(ctx context.Context, querier domain.Querier, accountID string, amount float64) error
}
