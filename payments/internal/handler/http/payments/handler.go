package payments_http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"payments/internal/app/payments"
	"payments/internal/domain"
)

type PaymentHandler struct {
	service payments.PaymentService
	logger  *zap.Logger
}

func NewPaymentHandler(s payments.PaymentService, l *zap.Logger) *PaymentHandler {
	return &PaymentHandler{service: s, logger: l}
}

type CreateUserAccountRequest struct {
	UserID  int64   `json:"user_id"`
	Balance float64 `json:"balance"`
}

type UpdateBalanceRequest struct {
	BalanceChange float64 `json:"balance_change"`
}

type BalanceResponse struct {
	UserID  int64   `json:"user_id"`
	Balance float64 `json:"balance"`
}

type UserAccountResponse struct {
	ID        string  `json:"id"`
	UserID    int64   `json:"user_id"`
	Balance   float64 `json:"balance"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func (h *PaymentHandler) CreateUserAccountWithBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Некорректное тело запроса для CreateUserAccount", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 {
		http.Error(w, "User ID is required and must be non-zero", http.StatusBadRequest)
		return
	}
	if req.Balance < 0 {
		http.Error(w, "Initial balance cannot be negative", http.StatusBadRequest)
		return
	}

	account, err := h.service.CreateAccount(r.Context(), req.UserID, req.Balance)
	if err != nil {
		if errors.Is(err, domain.ErrAccountAlreadyExists) {
			h.logger.Warn("Попытка создать существующий счет", zap.Int64("user_id", req.UserID))
			http.Error(w, "Account already exists for this user", http.StatusConflict)
			return
		}
		h.logger.Error("Не удалось создать счет пользователя", zap.Int64("user_id", req.UserID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := UserAccountResponse{
		ID:        account.ID,
		UserID:    account.UserID,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt.Format(http.TimeFormat),
		UpdatedAt: account.UpdatedAt.Format(http.TimeFormat),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("Не удалось отправить JSON-ответ", zap.Error(err))
	}
}

func (h *PaymentHandler) GetUserBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Некорректный формат User ID", zap.String("user_id_str", userIDStr), zap.Error(err))
		http.Error(w, "Invalid User ID format", http.StatusBadRequest)
		return
	}

	account, err := h.service.GetAccountForUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			h.logger.Warn("Счет не найден", zap.Int64("user_id", userID))
			http.Error(w, "Account not found for user", http.StatusNotFound)
			return
		}
		h.logger.Error("Не удалось получить баланс пользователя", zap.Int64("user_id", userID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := BalanceResponse{
		UserID:  account.UserID,
		Balance: account.Balance,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("Не удалось отправить JSON-ответ", zap.Error(err))
	}
}

func (h *PaymentHandler) UpdateUserBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("Некорректный формат User ID", zap.String("user_id_str", userIDStr), zap.Error(err))
		http.Error(w, "Invalid User ID format", http.StatusBadRequest)
		return
	}

	var req UpdateBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Некорректное тело запроса для UpdateUserBalance", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedAccount, err := h.service.UpdateUserAccountBalance(r.Context(), userID, req.BalanceChange)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			h.logger.Warn("Счет не найден для обновления баланса", zap.Int64("user_id", userID))
			http.Error(w, "Account not found for user", http.StatusNotFound)
			return
		}
		if errors.Is(err, domain.ErrInsufficientFunds) {
			h.logger.Warn("Недостаточно средств для списания", zap.Int64("user_id", userID), zap.Float64("balance_change", req.BalanceChange))
			http.Error(w, "Insufficient funds", http.StatusBadRequest)
			return
		}
		h.logger.Error("Не удалось обновить баланс пользователя", zap.Int64("user_id", userID), zap.Float64("balance_change", req.BalanceChange), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := BalanceResponse{
		UserID:  updatedAccount.UserID,
		Balance: updatedAccount.Balance,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("Не удалось отправить JSON-ответ", zap.Error(err))
	}
}
