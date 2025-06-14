package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/domain"
	"payments/internal/repository/accounts_repo"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) CreateAccountTx(ctx context.Context, tx *sql.Tx, account *domain.Account) error {
	query := `
		INSERT INTO accounts (id, user_id, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := tx.ExecContext(ctx, query,
		account.ID, account.UserID, account.Balance, account.CreatedAt, account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

func (r *AccountRepository) GetAccountForUserTx(ctx context.Context, tx *sql.Tx, userID string) (*domain.Account, error) {
	query := `
		SELECT id, user_id, balance, created_at, updated_at
		FROM accounts
		WHERE user_id = $1
		FOR UPDATE -- Важно: блокируем строку для изменения
	`
	account := &domain.Account{}
	err := tx.QueryRowContext(ctx, query, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, accounts_repo.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account for user %s: %w", userID, err)
	}
	return account, nil
}

func (r *AccountRepository) UpdateBalanceTx(ctx context.Context, tx *sql.Tx, accountID string, amount float64) error {
	query := `
		UPDATE accounts
		SET balance = balance + $1, updated_at = $2
		WHERE id = $3
	`
	if amount < 0 {
		checkBalanceQuery := `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`
		var currentBalance float64
		err := tx.QueryRowContext(ctx, checkBalanceQuery, accountID).Scan(&currentBalance)
		if err != nil {
			if err == sql.ErrNoRows {
				return accounts_repo.ErrAccountNotFound
			}
			return fmt.Errorf("failed to check current balance for account %s: %w", accountID, err)
		}
		if currentBalance+amount < 0 {
			return accounts_repo.ErrInsufficientFunds
		}
	}

	res, err := tx.ExecContext(ctx, query, amount, time.Now(), accountID)
	if err != nil {
		return fmt.Errorf("failed to update account balance for %s: %w", accountID, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return accounts_repo.ErrAccountNotFound
	}
	return nil
}
