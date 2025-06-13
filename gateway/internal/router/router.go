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

	orderProxy := createProxy(ordersURL, "/orders")
	paymentProxy := createProxy(paymentsURL, "")

	r.Group(func(r chi.Router) {
		r.Post("/users", paymentProxy.ServeHTTP)
		r.Patch("/users/{id}", paymentProxy.ServeHTTP)
		r.Get("/users/{id}", paymentProxy.ServeHTTP)
	})

	r.Group(func(r chi.Router) {
		r.Post("/orders", orderProxy.ServeHTTP)
		r.Get("/orders", orderProxy.ServeHTTP)
		r.Get("/orders/{id}", orderProxy.ServeHTTP)
		r.Get("/users/{id}/orders", orderProxy.ServeHTTP)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway активен!"))
	})

	return r, nil
}

func createProxy(target *url.URL, pathPrefixToStrip string) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme
		if pathPrefixToStrip != "" && strings.HasPrefix(req.URL.Path, pathPrefixToStrip) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, pathPrefixToStrip)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
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
