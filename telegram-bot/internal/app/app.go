package app

import (
	"context"
	"fmt"
	"log"

	chatclient "repairCopilotBot/chat-bot/pkg/client/chat"
	"repairCopilotBot/telegram-bot/config"
	"repairCopilotBot/telegram-bot/internal/repository"
	"repairCopilotBot/telegram-bot/internal/service"
	"repairCopilotBot/telegram-bot/internal/telegram"
	"repairCopilotBot/telegram-bot/pkg/database"
	userserviceclient "repairCopilotBot/user-service/client"
)

type App struct {
	config  *config.Config
	handler *telegram.Handler
}

func New(cfg *config.Config) (*App, error) {
	// Подключаемся к PostgreSQL
	db, err := database.NewPostgresConnection(database.PostgresConfig{
		Host:           cfg.Postgres.Host,
		Port:           cfg.Postgres.Port,
		DatabaseName:   cfg.Postgres.DatabaseName,
		Username:       cfg.Postgres.Username,
		Password:       cfg.Postgres.Password,
		MaxConnections: cfg.Postgres.MaxConnections,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Connected to PostgreSQL")

	// Создаем repository
	repo := repository.NewPostgresRepository(db)

	// Создаем клиент для user-service
	userClient, err := userserviceclient.NewUserClient(cfg.UserService.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create user service client: %w", err)
	}

	log.Println("Connected to user-service")

	// Создаем клиент для search-service
	chatClient, err := chatclient.New(&chatclient.Config{
		Address: cfg.ChatService.Address,
		Timeout: cfg.ChatService.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create search service client: %w", err)
	}

	log.Println("Connected to search-service")

	// Создаем service
	svc := service.NewService(repo, userClient, chatClient)

	// Создаем Telegram handler (передаем repo для state machine)
	handler, err := telegram.NewHandler(cfg.Telegram.BotToken, svc, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram handler: %w", err)
	}

	return &App{
		config:  cfg,
		handler: handler,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Println("Starting Telegram bot...")
	return a.handler.Start(ctx)
}
