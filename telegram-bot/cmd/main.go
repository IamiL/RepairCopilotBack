package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"repairCopilotBot/telegram-bot/config"
	"repairCopilotBot/telegram-bot/internal/app"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.MustLoad()

	// Создаем приложение
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обрабатываем сигналы для graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Запускаем приложение
	if err := application.Run(ctx); err != nil {
		if err != context.Canceled {
			log.Fatalf("App error: %v", err)
		}
	}

	log.Println("Telegram bot stopped")
}