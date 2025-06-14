package orders

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"orders/internal/app/orders"
)

func RegisterRoutes(r chi.Router, s orders.OrderService, l *zap.Logger) {
	handler := NewOrderHandler(s, l.With(zap.String("component", "OrderHTTPHandler")))

	r.Route("/health", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Orders service is healthy!"))
		})
	})

	r.Route("/orders", func(r chi.Router) {
		r.Post("/", handler.CreateOrder)
		r.Get("/", handler.GetAllOrders)
		r.Get("/{orderID}", handler.GetOrder)
		r.Get("/user/{userID}", handler.GetOrdersByUserID)
	})
}
