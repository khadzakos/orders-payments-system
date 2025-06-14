package router

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"gateway/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(cfg *config.Config) (http.Handler, error) {
	ordersURL, err := url.Parse(cfg.OrdersServiceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Order Service URL (%s): %w", cfg.OrdersServiceURL, err)
	}
	paymentsURL, err := url.Parse(cfg.PaymentsServiceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Payments Service URL (%s): %w", cfg.PaymentsServiceURL, err)
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	orderProxy := createProxy(ordersURL)
	paymentProxy := createProxy(paymentsURL)

	r.Route("/users", func(r chi.Router) {
		r.Get("/", paymentProxy.ServeHTTP)
		r.Post("/", paymentProxy.ServeHTTP)
		r.Patch("/{id}", paymentProxy.ServeHTTP)
		r.Get("/{id}", paymentProxy.ServeHTTP)
	})

	r.Route("/orders", func(r chi.Router) {
		r.Post("/", orderProxy.ServeHTTP)
		r.Get("/", orderProxy.ServeHTTP)
		r.Get("/{id}", orderProxy.ServeHTTP)
		r.Get("/user/{userID}", orderProxy.ServeHTTP)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway активен!"))
	})

	return r, nil
}

func createProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.RequestURI = req.URL.RequestURI()

		if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
			if prior, ok := req.Header["X-Forwarded-For"]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
			req.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error for request %s to %s: %v", r.URL.Path, target.String(), err)

		if os.IsTimeout(err) {
			renderJSONError(w, "Gateway Timeout", http.StatusGatewayTimeout)
		} else if _, ok := err.(net.Error); ok {
			renderJSONError(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			renderJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}

	return proxy
}
func renderJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s", "code": %d}`, message, statusCode)
}
