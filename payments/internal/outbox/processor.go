package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"payments/internal/domain"
	kafkaInfra "payments/internal/infrastucture/kafka" // Alias for kafka infrastructure producer interface

	"go.uber.org/zap"
)

// OutboxRepository defines the interface for interacting with the outbox message store.
// This allows for easier testing and decoupling from a specific repository implementation.
type OutboxRepository interface {
	GetPendingMessages(ctx context.Context, querier domain.Querier, limit int) ([]domain.OutboxMessage, error)
	UpdateMessageStatusTx(ctx context.Context, querier domain.Querier, id string, status domain.OutboxMessageStatus) error
}

type Processor struct {
	db             *sql.DB
	outboxRepo     OutboxRepository
	kafkaProducer  kafkaInfra.Producer // Use the interface from the infrastructure layer
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
	kafkaProducer kafkaInfra.Producer, // Use the interface from the infrastructure layer
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

// Start begins the outbox processing loop.
func (p *Processor) Start(ctx context.Context) {
	p.logger.Info("Starting outbox processor...")
	ticker := time.NewTicker(p.pollInterval) // Corrected: use pollInterval
	defer ticker.Stop()

	// Goroutine to signal when processing is truly done, for cleaner shutdown if needed by Stop()
	doneProcessing := make(chan struct{})

	go func() {
		defer close(doneProcessing)
		for {
			select {
			case <-ctx.Done(): // Main context cancelled (e.g. application shutdown)
				p.logger.Info("Outbox processor stopping due to main context cancellation.")
				return
			case <-p.shutdownSignal: // Explicit stop signal
				p.logger.Info("Outbox processor stopping due to explicit signal.")
				return
			case <-ticker.C:
				// Create a new context for this specific processing run to avoid using the long-lived Start context directly
				// This is important if processOutboxMessages itself uses timeouts based on its input context.
				// However, processOutboxMessages already creates its own timeout context for DB operations.
				p.processOutboxMessages(ctx) // Pass the main context, which processOutboxMessages will use to derive its own short-lived contexts
			}
		}
	}()

	// Wait for either the main context to be done or the explicit shutdown signal
	// This keeps Start blocking until the processor is meant to stop, aligning with how main.go might expect it.
	// However, main.go starts it in a goroutine and relies on ctxMain for shutdown signal.
	// So, this Start method should probably not block here if it's started as `go outboxProcessor.Start(ctxMain)`
	// For now, let's assume Start is meant to be a blocking call that the caller can run in a goroutine.
	// If main.go handles the goroutine and shutdown via ctxMain, then this waiting logic here might be redundant
	// or could be simplified. The current main.go structure suggests this Start method's goroutine will exit
	// when ctxMain is cancelled.

	// The Stop() method will close p.shutdownSignal. The goroutine above will see that and exit.
	// If Start is run in its own goroutine by the caller, then Stop() might want to wait on `doneProcessing`.
}

// Stop signals the outbox processor to stop.
// It's idempotent and safe to call multiple times.
func (p *Processor) Stop() {
	p.shutdownOnce.Do(func() {
		p.logger.Info("Signaling outbox processor to stop...")
		close(p.shutdownSignal)
	})
	// Note: This Stop() does not wait for the processor to fully finish.
	// main.go relies on cancelling ctxMain and then gives some time for goroutines to finish.
	// If a more guaranteed stop is needed, Start could return a channel that Stop waits on.
	// Or, the goroutine in Start could signal its completion on a channel that Stop waits for.
	// For now, aligning with main.go's shutdown logic where it cancels a context and waits.
	p.logger.Info("Outbox processor stop signal sent.")
}

func (p *Processor) processOutboxMessages(ctx context.Context) {
	p.logger.Debug("Polling for outbox messages...")

	// Get a batch of pending messages. The limit can be adjusted.
	// We use a timeout for the database query to prevent it from blocking indefinitely.
	// The input ctx is the parent context for the processor's run.
	dbQueryCtx, cancel := context.WithTimeout(ctx, p.pollTimeout) // Corrected: use the passed ctx as parent
	defer cancel()

	messages, err := p.outboxRepo.GetPendingMessages(dbQueryCtx, p.db, 10) // Example limit
	if err != nil {
		if err == sql.ErrNoRows {
			// No pending messages, this is normal
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
		// Create a new context for this transaction, possibly with a timeout.
		// Using the dbQueryCtx's parent (which is the processor's main ctx) for the transaction itself.
		txCtx := ctx // Or context.Background() if we want it fully independent of the processor's lifecycle for this iteration.
		// Using ctx (processor's main context) means if processor is stopping, new txns might not start or might be cancelled.

		tx, err := p.db.BeginTx(txCtx, nil)
		if err != nil {
			p.logger.Error("Failed to begin transaction for outbox message", zap.String("message_id", msg.ID), zap.Error(err))
			continue // Try next message
		}

		// 1. Send to Kafka
		// The context for Kafka produce operation should also be short-lived if possible.
		// For now, using the transaction context `txCtx` which is derived from the processor's main context `ctx`.
		// If `p.kafkaProducer.Produce` is blocking for too long, this could be an issue.
		// Consider creating a separate, short-lived context for the Produce call if necessary.
		if err := p.kafkaProducer.Produce(txCtx, p.topic, msg.Payload); err != nil {
			p.logger.Error("Failed to send message to Kafka",
				zap.String("message_id", msg.ID),
				zap.String("topic", p.topic),
				zap.Error(err))
			tx.Rollback()
			continue
		}
		p.logger.Info("Message sent to Kafka successfully", zap.String("message_id", msg.ID), zap.String("topic", p.topic))

		// 2. Update message status in outbox
		if err := p.outboxRepo.UpdateMessageStatusTx(ctx, tx, msg.ID, domain.OutboxStatusSent); err != nil {
			p.logger.Error("Failed to update outbox message status to SENT", zap.String("message_id", msg.ID), zap.Error(err))
			tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			p.logger.Error("Failed to commit transaction for outbox message", zap.String("message_id", msg.ID), zap.Error(err))
			// If commit fails, Kafka message was sent but DB status not updated.
			// This could lead to duplicate sends if not handled carefully (e.g. by consumer idempotency).
			// For now, we log and continue. The message will be picked up again if status wasn't updated.
			continue
		}

		p.logger.Info("Outbox message processed and status updated", zap.String("message_id", msg.ID))
	}
}

// PaymentStatusUpdateEvent represents the structure of the message sent to Kafka for payment status updates.
type PaymentStatusUpdateEvent struct {
	PaymentID string    `json:"payment_id,omitempty"`
	OrderID   string    `json:"order_id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"` // e.g., COMPLETED, FAILED
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"` // e.g., insufficient_funds
}

// Helper to marshal payload for outbox, used in payment_service.go
func PreparePaymentStatusUpdatePayload(paymentID, orderID string, userID int64, amount float64, status domain.PaymentStatus, eventTime time.Time, errMsg string) ([]byte, error) {
	event := PaymentStatusUpdateEvent{
		PaymentID: paymentID,
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Status:    string(status),
		Timestamp: eventTime,
		Error:     errMsg,
	}
	return json.Marshal(event)
}
