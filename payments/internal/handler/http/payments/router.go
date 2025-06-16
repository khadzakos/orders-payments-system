package payments_http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"go.uber.org/zap"

	"payments/internal/app/payments"
)

func RegisterRoutes(r chi.Router, s payments.PaymentService, l *zap.Logger) {
	handler := NewPaymentHandler(s, l.With(zap.String("component", "PaymentHTTPHandler")))

	r.Route("/health", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Payments service is healthy!"))
		})
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/", handler.CreateUserAccountWithBalanceHandler)
		r.Get("/{id}", handler.GetUserBalanceHandler)
		r.Patch("/{id}", handler.UpdateUserBalanceHandler)
	})

}
