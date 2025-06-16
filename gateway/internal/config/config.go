package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GatewayPort        int
	OrdersServiceURL   string
	PaymentsServiceURL string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	gatewayPortStr := os.Getenv("GATEWAY_PORT")
	if gatewayPortStr == "" {
		gatewayPortStr = "80"
	}
	port, err := strconv.Atoi(gatewayPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid GATEWAY_PORT: %w", err)
	}
	cfg.GatewayPort = port

	cfg.OrdersServiceURL = os.Getenv("ORDERS_SERVICE_HOST")
	if cfg.OrdersServiceURL == "" {
		cfg.OrdersServiceURL = "http://localhost:8081"
	}

	cfg.PaymentsServiceURL = os.Getenv("PAYMENTS_SERVICE_HOST")
	if cfg.PaymentsServiceURL == "" {
		cfg.PaymentsServiceURL = "http://localhost:8082"
	}

	return cfg, nil
}
