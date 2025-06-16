package accounts_repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/domain"

	"github.com/lib/pq"
)

type accountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *accountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) CreateAccountTx(ctx context.Context, querier domain.Querier, account *domain.Account) error {
	query := `
		INSERT INTO accounts (id, user_id, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := querier.ExecContext(ctx, query,
		account.ID, account.UserID, account.Balance, account.CreatedAt, account.UpdatedAt)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return domain.ErrAccountAlreadyExists
		}
		return fmt.Errorf("failed to create account for user %d: %w", account.UserID, err)
	}
	return nil
}

func (r *accountRepository) GetAccountForUserTx(ctx context.Context, querier domain.Querier, userID int64) (*domain.Account, error) {
	query := `
		SELECT id, user_id, balance, created_at, updated_at
		FROM accounts
		WHERE user_id = $1
		FOR UPDATE
	`
	account := &domain.Account{}
	err := querier.QueryRowContext(ctx, query, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account for user %d: %w", userID, err)
	}
	return account, nil
}

func (r *accountRepository) UpdateBalanceTx(ctx context.Context, querier domain.Querier, accountID string, amount float64) error {
	checkBalanceQuery := `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`
	var currentBalance float64
	err := querier.QueryRowContext(ctx, checkBalanceQuery, accountID).Scan(&currentBalance)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ErrAccountNotFound
		}
		return fmt.Errorf("failed to check current balance for account %s: %w", accountID, err)
	}
	if currentBalance+amount < 0 {
		return domain.ErrInsufficientFunds
	}

	query := `
		UPDATE accounts
		SET balance = balance + $1, updated_at = $2
		WHERE id = $3
	`
	res, err := querier.ExecContext(ctx, query, amount, time.Now(), accountID)
	if err != nil {
		return fmt.Errorf("failed to update account balance for %s: %w", accountID, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("account with id %s not found for update", accountID)
	}
	return nil
}
