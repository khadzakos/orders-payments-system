package main

import (
	"fmt"
	"gateway/internal/config"
	"gateway/internal/router"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию: %v", err)
	}

	r, err := router.NewRouter(cfg)
	if err != nil {
		log.Fatalf("Не удалось создать маршрутизатор: %v", err)
	}

	addr := fmt.Sprintf(":%d", cfg.GatewayPort)
	log.Printf("Запуск API Gateway на порту %d", cfg.GatewayPort)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
