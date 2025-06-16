package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"payments/internal/domain"
	kafkaInfra "payments/internal/infrastucture/kafka"

	"go.uber.org/zap"
)

type OutboxRepository interface {
	GetPendingMessages(ctx context.Context, querier domain.Querier, limit int) ([]domain.OutboxMessage, error)
	UpdateMessageStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.OutboxMessageStatus) error
}

type Processor struct {
	db             *sql.DB
	outboxRepo     OutboxRepository
	kafkaProducer  kafkaInfra.Producer
	topic          string
	pollInterval   time.Duration
	pollTimeout    time.Duration
	logger         *zap.Logger
	shutdownSignal chan struct{}
	shutdownOnce   sync.Once
}

func NewProcessor(
	db *sql.DB,
	outboxRepo OutboxRepository,
	kafkaProducer kafkaInfra.Producer,
	topic string,
	pollInterval time.Duration,
	pollTimeout time.Duration,
	logger *zap.Logger,
) *Processor {
	return &Processor{
		db:             db,
		outboxRepo:     outboxRepo,
		kafkaProducer:  kafkaProducer,
		topic:          topic,
		pollInterval:   pollInterval,
		pollTimeout:    pollTimeout,
		logger:         logger,
		shutdownSignal: make(chan struct{}),
	}
}

func (p *Processor) Start(ctx context.Context) {
	p.logger.Info("Starting outbox processor...")
	ticker := time.NewTicker(p.pollInterval)

	doneProcessing := make(chan struct{})

	go func() {
		defer close(doneProcessing)
		for {
			select {
			case <-ticker.C:
				p.processOutboxMessages(ctx)
			}
		}
	}()
}

func (p *Processor) Stop() {
	p.shutdownOnce.Do(func() {
		p.logger.Info("Signaling outbox processor to stop...")
		close(p.shutdownSignal)
	})
	p.logger.Info("Outbox processor stop signal sent.")
}

func (p *Processor) processOutboxMessages(ctx context.Context) {
	p.logger.Debug("Polling for outbox messages...")

	dbQueryCtx, cancel := context.WithTimeout(ctx, p.pollTimeout)
	defer cancel()

	messages, err := p.outboxRepo.GetPendingMessages(dbQueryCtx, p.db, 10)
	if err != nil {
		if err == sql.ErrNoRows {
			return
		}
		p.logger.Error("Failed to get pending outbox messages", zap.Error(err))
		return
	}

	if len(messages) == 0 {
		p.logger.Debug("No pending outbox messages found.")
		return
	}

	p.logger.Info("Found pending outbox messages", zap.Int("count", len(messages)))

	for _, msg := range messages {
		txCtx := ctx

		tx, err := p.db.BeginTx(txCtx, nil)
		if err != nil {
			p.logger.Error("Failed to begin transaction for outbox message", zap.String("message_id", msg.ID), zap.Error(err))
			continue
		}

		if err := p.kafkaProducer.Produce(txCtx, p.topic, msg.Payload); err != nil {
			p.logger.Error("Failed to send message to Kafka",
				zap.String("message_id", msg.ID),
				zap.String("topic", p.topic),
				zap.Error(err))
			tx.Rollback()
			continue
		}
		p.logger.Info("Message sent to Kafka successfully", zap.String("message_id", msg.ID), zap.String("topic", p.topic))

		if err := p.outboxRepo.UpdateMessageStatusTx(ctx, tx, msg.ID, domain.OutboxStatusSent); err != nil {
			p.logger.Error("Failed to update outbox message status to SENT", zap.String("message_id", msg.ID), zap.Error(err))
			tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			p.logger.Error("Failed to commit transaction for outbox message", zap.String("message_id", msg.ID), zap.Error(err))
			continue
		}

		p.logger.Info("Outbox message processed and status updated", zap.String("message_id", msg.ID))
	}
}

type PaymentStatusUpdateEvent struct {
	PaymentID string    `json:"payment_id,omitempty"`
	OrderID   string    `json:"order_id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

func PreparePaymentStatusUpdatePayload(paymentID, orderID string, userID int64, amount float64, status domain.PaymentStatus, eventTime time.Time, errMsg string) ([]byte, error) {
	var statusStr string
	switch status {
	case domain.PaymentStatusCompleted:
		statusStr = "SUCCESS"
	case domain.PaymentStatusFailed:
		statusStr = "FAILED"
	default:
		statusStr = string(status)
	}

	event := PaymentStatusUpdateEvent{
		PaymentID: paymentID,
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Status:    statusStr,
		Timestamp: eventTime,
		Error:     errMsg,
	}
	return json.Marshal(event)
}
