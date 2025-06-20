package orders

type CreateOrderRequest struct {
	UserID      int64   `json:"user_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

type OrderResponse struct {
	ID          string  `json:"id"`
	UserID      int64   `json:"user_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
}

type PaymentStatusEvent struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}
