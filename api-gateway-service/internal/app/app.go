package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	httpapp "repairCopilotBot/api-gateway-service/internal/app/http"
	tgClient "repairCopilotBot/api-gateway-service/internal/pkg/tg"
	"repairCopilotBot/api-gateway-service/internal/repository"
	userserviceclient "repairCopilotBot/user-service/client"
	tzbotclient "repairCopilotBot/tz-bot/client"
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
	UserServiceAddr string,
) *App {
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

	sessionRepo := repository.NewSessionRepository(RedisConfig.Address, RedisConfig.Password)

	tgBot, err := tgClient.NewBot(TgConfig.Token)
	if err != nil {
		panic(err)
	}

	tgClient.New(tgBot, TgConfig.ChatID)

	httpApp := httpapp.New(log, httpConfig, tzBotClient, userServiceClient, sessionRepo)

	return &App{
		HTTPServer: httpApp,
	}
}
