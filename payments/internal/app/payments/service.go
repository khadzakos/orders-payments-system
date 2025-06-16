package payments

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payments/internal/outbox"
	"payments/internal/repository/accounts_repo"
	"payments/internal/repository/inbox_repo"
	"payments/internal/repository/outbox_repo"
	"payments/internal/repository/payments_repo"
	"payments/internal/util"

	"go.uber.org/zap"

	"payments/internal/domain"
)

type PaymentService interface {
	ProcessOrderPayment(ctx context.Context, orderID string, userID int64, amount float64) (*domain.Payment, error)
	CreateAccount(ctx context.Context, userID int64, initialBalance float64) (*domain.Account, error)
	GetAccountForUser(ctx context.Context, userID int64) (*domain.Account, error)
	UpdateUserAccountBalance(ctx context.Context, userID int64, amountChange float64) (*domain.Account, error)
	ProcessIncomingOrderCreatedEvent(ctx context.Context, eventID string, orderID string, userID int64, amount float64, rawPayload []byte) error
}

type paymentService struct {
	db          *sql.DB
	accountRepo accounts_repo.AccountRepository
	paymentRepo payments_repo.PaymentRepository
	inboxRepo   inbox_repo.InboxRepository
	outboxRepo  outbox_repo.OutboxRepository
	logger      *zap.Logger
}

func NewPaymentService(
	db *sql.DB,
	accountRepo accounts_repo.AccountRepository,
	paymentRepo payments_repo.PaymentRepository,
	inboxRepo inbox_repo.InboxRepository,
	outboxRepo outbox_repo.OutboxRepository,
	logger *zap.Logger,
) PaymentService {
	return &paymentService{
		db:          db,
		accountRepo: accountRepo,
		paymentRepo: paymentRepo,
		inboxRepo:   inboxRepo,
		outboxRepo:  outboxRepo,
		logger:      logger,
	}
}

func (s *paymentService) ProcessOrderPayment(ctx context.Context, orderID string, userID int64, amount float64) (*domain.Payment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Не удалось начать транзакцию для платежа по заказу", zap.String("order_id", orderID), zap.Error(err))
		return nil, fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Восстановлена паника во время транзакции платежа по заказу, выполняется откат", zap.String("order_id", orderID), zap.Any("panic", r))
			tx.Rollback()
			panic(r)
		}
	}()

	payment, err := s.processOrderPaymentTx(ctx, tx, orderID, userID, amount)
	if err != nil {
		s.logger.Error("Не удалось обработать платеж по заказу, выполняется откат транзакции", zap.String("order_id", orderID), zap.Error(err))
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logger.Error("Не удалось откатить транзакцию", zap.String("order_id", orderID), zap.Error(rbErr))
			return nil, fmt.Errorf("откат транзакции завершился неудачей после ошибки обработки: %w", rbErr)
		}
		return nil, fmt.Errorf("не удалось обработать платеж по заказу: %w", err)
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("Не удалось зафиксировать транзакцию для платежа по заказу", zap.String("order_id", orderID), zap.Error(err))
		return nil, fmt.Errorf("не удалось зафиксировать транзакцию: %w", err)
	}

	s.logger.Info("Платеж по заказу успешно обработан", zap.String("payment_id", payment.ID), zap.String("order_id", orderID), zap.Float64("amount", amount))
	return payment, nil
}

func (s *paymentService) processOrderPaymentTx(ctx context.Context, tx *sql.Tx, orderID string, userID int64, amount float64) (*domain.Payment, error) {
	existingPayment, err := s.paymentRepo.GetByOrderIDTx(ctx, tx, orderID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("не удалось проверить существующий платеж для заказа %s: %w", orderID, err)
	}
	if existingPayment != nil {
		s.logger.Info("Платеж уже обработан для заказа",
			zap.String("order_id", orderID),
			zap.String("payment_id", existingPayment.ID),
			zap.String("status", string(existingPayment.Status)))
		return existingPayment, fmt.Errorf("платеж для заказа %s уже существует со статусом %s", orderID, existingPayment.Status)
	}

	userAccountID := userID
	account, err := s.accountRepo.GetAccountForUserTx(ctx, tx, userAccountID)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			s.logger.Warn("Счет для пользователя не найден", zap.Int64("user_id", userAccountID))
			return nil, fmt.Errorf("счет для пользователя %d не найден: %w", userAccountID, err)
		} else {
			return nil, fmt.Errorf("не удалось получить счет для пользователя %d: %w", userAccountID, err)
		}
	}

	deductionAmount := -amount
	if err = s.accountRepo.UpdateBalanceTx(ctx, tx, account.ID, deductionAmount); err != nil {
		if err == domain.ErrInsufficientFunds {
			s.logger.Warn("Недостаточно средств для платежа", zap.String("order_id", orderID), zap.Float64("amount", amount), zap.String("account_id", account.ID))
			paymentID := util.GenerateUUID()
			failedPayment := &domain.Payment{
				ID:        paymentID,
				OrderID:   orderID,
				UserID:    userID,
				Amount:    amount,
				Status:    domain.PaymentStatusFailed,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if createErr := s.paymentRepo.CreateTx(ctx, tx, failedPayment); createErr != nil {
				s.logger.Error("Не удалось создать запись о проваленном платеже", zap.String("order_id", orderID), zap.Error(createErr))
			}

			// Prepare payload for outbox message indicating failed payment
			failedPaymentTime := time.Now() // Use consistent time for event
			payloadBytes, err := outbox.PreparePaymentStatusUpdatePayload(
				failedPayment.ID, // Use the ID of the created failed payment record
				orderID,
				userID,
				amount,
				domain.PaymentStatusFailed,
				failedPaymentTime,
				"insufficient_funds",
			)
			if err != nil {
				s.logger.Error("Не удалось подготовить payload для outbox (проваленный платеж)", zap.String("order_id", orderID), zap.Error(err))
				// Continue to attempt creating outbox message even if payload prep fails, maybe with generic error
			}

			outboxMsg := &domain.OutboxMessage{
				ID:          util.GenerateUUID(),
				OrderID:     orderID,
				OrderStatus: string(domain.PaymentStatusFailed), // This might need adjustment based on how orders service interprets this
				Payload:     payloadBytes,
				Status:      domain.OutboxStatusPending,
				CreatedAt:   failedPaymentTime, // Use consistent time
			}
			if outboxErr := s.outboxRepo.CreateMessageTx(ctx, tx, outboxMsg); outboxErr != nil {
				s.logger.Error("Не удалось создать сообщение outbox для проваленного платежа", zap.String("order_id", orderID), zap.Error(outboxErr))
			}

			return nil, domain.ErrInsufficientFunds
		}
		return nil, fmt.Errorf("не удалось обновить баланс счета для пользователя %d: %w", userID, err)
	}

	paymentID := util.GenerateUUID()
	now := time.Now()
	payment := &domain.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Status:    domain.PaymentStatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.paymentRepo.CreateTx(ctx, tx, payment); err != nil {
		return nil, fmt.Errorf("не удалось создать запись о платеже для заказа %s: %w", orderID, err)
	}

	payloadBytes, err := outbox.PreparePaymentStatusUpdatePayload(
		payment.ID,
		orderID,
		userID,
		amount,
		domain.PaymentStatusCompleted,
		now,
		"",
	)
	if err != nil {
		s.logger.Error("Не удалось подготовить payload для outbox (успешный платеж)", zap.String("order_id", orderID), zap.Error(err))
	}

	outboxMsg := &domain.OutboxMessage{
		ID:          util.GenerateUUID(),
		OrderID:     orderID,
		OrderStatus: string(domain.PaymentStatusCompleted),
		Payload:     payloadBytes,
		Status:      domain.OutboxStatusPending,
		CreatedAt:   now,
	}
	if err := s.outboxRepo.CreateMessageTx(ctx, tx, outboxMsg); err != nil {
		return nil, fmt.Errorf("не удалось создать outbox сообщение для заказа %s: %w", orderID, err)
	}

	return payment, nil
}

func (s *paymentService) CreateAccount(ctx context.Context, userID int64, initialBalance float64) (*domain.Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Не удалось начать транзакцию для создания счета", zap.Int64("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	existingAccount, err := s.accountRepo.GetAccountForUserTx(ctx, tx, userID)
	if err != nil && err != domain.ErrAccountNotFound {
		tx.Rollback()
		return nil, fmt.Errorf("не удалось проверить существующий счет: %w", err)
	}
	if existingAccount != nil {
		s.logger.Warn("Счет уже существует для пользователя", zap.Int64("user_id", userID), zap.String("account_id", existingAccount.ID))
		tx.Rollback()
		return existingAccount, domain.ErrAccountAlreadyExists
	}

	accountID := util.GenerateUUID()
	now := time.Now()
	newAccount := &domain.Account{
		ID:        accountID,
		UserID:    userID,
		Balance:   initialBalance,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.accountRepo.CreateAccountTx(ctx, tx, newAccount); err != nil {
		tx.Rollback()
		if err == domain.ErrAccountAlreadyExists {
			return nil, domain.ErrAccountAlreadyExists
		}
		return nil, fmt.Errorf("не удалось создать счет для пользователя %d: %w", userID, err)
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("Не удалось зафиксировать транзакцию для создания счета", zap.Int64("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("не удалось зафиксировать транзакцию: %w", err)
	}

	s.logger.Info("Счет успешно создан", zap.String("account_id", newAccount.ID), zap.Int64("user_id", userID), zap.Float64("balance", initialBalance))
	return newAccount, nil
}

func (s *paymentService) GetAccountForUser(ctx context.Context, userID int64) (*domain.Account, error) {
	account, err := s.accountRepo.GetAccountForUserTx(ctx, s.db, userID)
	if err != nil {
		s.logger.Warn("Не удалось получить счет для пользователя", zap.Int64("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("не удалось получить счет для пользователя %d: %w", userID, err)
	}
	return account, nil
}

func (s *paymentService) UpdateUserAccountBalance(ctx context.Context, userID int64, amountChange float64) (*domain.Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Не удалось начать транзакцию для обновления баланса", zap.Int64("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	account, err := s.accountRepo.GetAccountForUserTx(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		if err == domain.ErrAccountNotFound {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("не удалось получить счет для пользователя %d для обновления баланса: %w", userID, err)
	}

	if account.Balance+amountChange < 0 {
		tx.Rollback()
		return nil, domain.ErrInsufficientFunds
	}

	if err := s.accountRepo.UpdateBalanceTx(ctx, tx, account.ID, amountChange); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("не удалось обновить баланс для счета %s (пользователь %d): %w", account.ID, userID, err)
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("Не удалось зафиксировать транзакцию для обновления баланса", zap.Int64("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("не удалось зафиксировать транзакцию: %w", err)
	}

	s.logger.Info("Баланс счета пользователя успешно обновлен",
		zap.Int64("user_id", userID),
		zap.String("account_id", account.ID),
		zap.Float64("old_balance", account.Balance),
		zap.Float64("amount_change", amountChange),
		zap.Float64("new_balance", account.Balance+amountChange))

	updatedAccount, err := s.accountRepo.GetAccountForUserTx(ctx, s.db, userID)
	if err != nil {
		s.logger.Error("Не удалось получить обновленный счет после изменения баланса", zap.Int64("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("не удалось получить обновленный счет: %w", err)
	}

	return updatedAccount, nil
}

func (s *paymentService) ProcessIncomingOrderCreatedEvent(ctx context.Context, eventID string, orderID string, userID int64, amount float64, rawPayload []byte) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Не удалось начать транзакцию для обработки inbox", zap.String("event_id", eventID), zap.Error(err))
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Восстановлена паника во время транзакции обработки inbox, выполняется откат", zap.String("event_id", eventID), zap.Any("panic", r))
			tx.Rollback()
			panic(r)
		}
	}()

	inboxMsg := &domain.InboxMessage{
		ID:         eventID,
		OrderID:    orderID,
		Payload:    rawPayload,
		Status:     domain.InboxStatusNew,
		ReceivedAt: time.Now(),
	}

	if err = s.inboxRepo.CreateMessageTx(ctx, tx, inboxMsg); err != nil {
		if err == domain.ErrMessageAlreadyProcessed {
			s.logger.Info("Входящее OrderCreatedEvent уже обработано (проверка идемпотентности)", zap.String("event_id", eventID), zap.String("order_id", orderID))
			if rbErr := tx.Rollback(); rbErr != nil {
				s.logger.Error("Не удалось откатить транзакцию после дубликата сообщения inbox", zap.String("event_id", eventID), zap.Error(rbErr))
			}
			return nil
		}
		s.logger.Error("Не удалось создать сообщение inbox", zap.String("event_id", eventID), zap.Error(err))
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logger.Error("Не удалось откатить транзакцию после ошибки создания сообщения inbox", zap.String("event_id", eventID), zap.Error(rbErr))
		}
		return fmt.Errorf("не удалось записать входящее событие: %w", err)
	}
	s.logger.Info("Сообщение inbox успешно записано", zap.String("event_id", eventID), zap.String("order_id", orderID))

	payment, err := s.processOrderPaymentTx(ctx, tx, orderID, userID, amount)
	if err != nil {
		s.logger.Error("Не удалось обработать платеж из OrderCreatedEvent, помечаем inbox как проваленный", zap.String("event_id", eventID), zap.String("order_id", orderID), zap.Error(err))
		if updateErr := s.inboxRepo.UpdateStatusTx(ctx, tx, eventID, domain.InboxStatusFailed); updateErr != nil {
			s.logger.Error("Не удалось обновить статус сообщения inbox на FAILED", zap.String("event_id", eventID), zap.Error(updateErr))
		}
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logger.Error("Не удалось откатить транзакцию после сбоя обработки платежа", zap.String("event_id", eventID), zap.Error(rbErr))
		}
		return fmt.Errorf("не удалось обработать платеж для заказа %s из события %s: %w", orderID, eventID, err)
	}

	s.logger.Info("Платеж успешно обработан", zap.String("event_id", eventID), zap.String("order_id", orderID), zap.String("payment_id", payment.ID))

	if err := s.inboxRepo.UpdateStatusTx(ctx, tx, eventID, domain.InboxStatusCompleted); err != nil {
		s.logger.Error("Не удалось обновить статус сообщения inbox на COMPLETED", zap.String("event_id", eventID), zap.Error(err))
		if rbErr := tx.Rollback(); rbErr != nil {
			s.logger.Error("Не удалось откатить транзакцию после ошибки обновления статуса inbox", zap.String("event_id", eventID), zap.Error(rbErr))
		}
		return fmt.Errorf("не удалось пометить событие %s как обработанное: %w", eventID, err)
	}

	s.logger.Info("Сообщение inbox успешно помечено как обработанное", zap.String("event_id", eventID), zap.String("order_id", orderID))

	if err := tx.Commit(); err != nil {
		s.logger.Error("Не удалось зафиксировать транзакцию для обработки inbox", zap.String("event_id", eventID), zap.Error(err))
		return fmt.Errorf("не удалось зафиксировать транзакцию: %w", err)
	}

	s.logger.Info("OrderCreatedEvent успешно обработан, платеж завершен",
		zap.String("event_id", eventID),
		zap.String("order_id", orderID),
		zap.String("payment_id", payment.ID),
	)
	return nil
}
