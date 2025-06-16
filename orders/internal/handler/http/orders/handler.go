package orders

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"orders/internal/app/orders"
)

type OrderHandler struct {
	service orders.OrderService
	logger  *zap.Logger
}

func NewOrderHandler(s orders.OrderService, l *zap.Logger) *OrderHandler {
	return &OrderHandler{service: s, logger: l}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req orders.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body for CreateOrder", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	res, err := h.service.CreateOrder(r.Context(), &req)
	if err != nil {
		if errors.Is(err, orders.ErrInvalidOrder) {
			h.logger.Warn("Bad request for CreateOrder", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.logger.Error("Error creating order", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	if orderID == "" {
		h.logger.Warn("Order ID is missing in GetOrder request")
		http.Error(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	res, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			h.logger.Info("Order not found", zap.String("order_id", orderID))
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Error getting order", zap.String("order_id", orderID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *OrderHandler) GetOrdersByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userID")

	if userIDStr == "" {
		h.logger.Warn("User ID is missing in GetOrdersByUserID request")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("Invalid User ID in GetOrdersByUserID request", zap.Error(err))
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	res, err := h.service.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Error("Error getting orders for user", zap.Int64("user_id", userID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	res, err := h.service.GetAllOrders(r.Context())
	if err != nil {
		h.logger.Error("Error getting all orders", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
