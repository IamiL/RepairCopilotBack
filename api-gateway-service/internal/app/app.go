package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	httpapp "repairCopilotBot/api-gateway-service/internal/app/http"
	tgClient "repairCopilotBot/api-gateway-service/internal/pkg/tg"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/api-gateway-service/internal/repository/postgres"
	postgresActionLog "repairCopilotBot/api-gateway-service/internal/repository/postgres/action_log"
	chatBotClient "repairCopilotBot/chat-bot/pkg/client"
	searchBotClient "repairCopilotBot/search-bot/pkg/client"
	tzbotclient "repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"
	"time"
)

type Config struct {
	TokenTTL time.Duration `yaml:"token_ttl" env-default:"300h"`
}

type App struct {
	HTTPServer *httpapp.App
}

func New(
	log *slog.Logger,
	appConfig *Config,
	httpConfig *httpapp.Config,
	TgConfig *tgClient.Config,
	TzBotClientConfig *tzbotclient.Config,
	RedisConfig *repository.RedisConfig,
	PostgresConfig *postgres.Config,
	ChatBotClientConfig *chatBotClient.Config,
	SearchBotClientConfig *searchBotClient.Config,
	UserServiceAddr string,
) *App {
	postgresConn, err := postgres.NewConnPool(PostgresConfig)
	if err != nil {
		log.Error(fmt.Sprintf("error connect to postgres - %w", err))
		os.Exit(1)
	}

	actionLogRepo, err := postgresActionLog.New(postgresConn)
	if err != nil {
		log.Error(fmt.Sprintf("error init action log repo - %w", err))
		os.Exit(1)
	}

	tzBotClient, err := tzbotclient.New(context.Background(), TzBotClientConfig.Addr)
	if err != nil {
		log.Error(fmt.Sprintf("error connect to tzBot - %w", err))
		os.Exit(1)
	}

	userServiceClient, err := userserviceclient.NewUserClient(UserServiceAddr)
	if err != nil {
		log.Error(fmt.Sprintf("error connect to user service - %w", err))
		os.Exit(1)
	}

	chatBotClient, err := chatBotClient.New(ChatBotClientConfig)

	searchBotClient, err := searchBotClient.New(SearchBotClientConfig)

	sessionRepo := repository.NewSessionRepository(RedisConfig.Address, RedisConfig.Password)

	tgBot, err := tgClient.NewBot(TgConfig.Token)
	if err != nil {
		panic(err)
	}

	tgClient.New(tgBot, TgConfig.ChatID)

	httpApp := httpapp.New(log, httpConfig, tzBotClient, userServiceClient, chatBotClient, searchBotClient, sessionRepo, actionLogRepo)

	return &App{
		HTTPServer: httpApp,
	}
}
