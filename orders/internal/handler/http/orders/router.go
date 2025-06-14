package orders

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"orders/internal/app/orders"
)

func RegisterRoutes(r chi.Router, s orders.OrderService, l *zap.Logger) {
	handler := NewOrderHandler(s, l.With(zap.String("component", "OrderHTTPHandler")))

	r.Route("/orders", func(r chi.Router) {
		r.Post("/", handler.CreateOrder)
		r.Get("/", handler.GetAllOrders)
		r.Get("/{orderID}", handler.GetOrder)
		r.Get("/user/{userID}", handler.GetOrdersByUserID)
	})
}
