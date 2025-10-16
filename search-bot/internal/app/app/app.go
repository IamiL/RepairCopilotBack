package app

import (
	"log/slog"
	llmClient "repairCopilotBot/search-bot/internal/pkg/llmClient"

	"repairCopilotBot/search-bot/internal/app/grpc/server"
	"repairCopilotBot/search-bot/internal/repository/postgres"
	chatservice "repairCopilotBot/search-bot/internal/service/chat"
	userservice "repairCopilotBot/search-bot/internal/service/user"
)

type App struct {
	GRPCServer *server.ChatGRPCServer
}

func New(
	log *slog.Logger,
	grpcConfig *server.Config,
	postgresConfig *postgres.Config,
	llmClientConfig *llmClient.Config,
) *App {
	postgresConn, err := postgres.NewConnPool(postgresConfig)
	if err != nil {
		panic(err)
	}

	repository := postgres.New(postgresConn)

	llmClient, err := llmClient.New(*llmClientConfig)

	chatService := chatservice.New(log, repository, repository, repository, repository, repository, repository, llmClient)
	userService := userservice.New(log, repository)

	grpcApp := server.NewChatGRPCServer(log, chatService, userService, grpcConfig)

	return &App{
		GRPCServer: grpcApp,
	}
}
