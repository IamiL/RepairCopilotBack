package config

import (
	"repairCopilotBot/api-gateway-service/internal/app"
	httpapp "repairCopilotBot/api-gateway-service/internal/app/http"
	"repairCopilotBot/api-gateway-service/internal/pkg/tg"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/api-gateway-service/internal/repository/postgres"
	chatBotServiceClient "repairCopilotBot/chat-bot/pkg/client"
	searchBotServiceClient "repairCopilotBot/search-bot/pkg/client"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env              string                        `env:"ENV" env-default:"local"`
	App              app.Config                    `env-prefix:"APP_"`
	HTTP             httpapp.Config                `env-prefix:"HTTP_"`
	Tg               tg_client.Config              `env-prefix:"TG_"`
	TzBotService     client.Config                 `env-prefix:"TZ_SERVICE_"`
	ChatBotService   chatBotServiceClient.Config   `env-prefix:"CHAT_POCHEMU_SERVICE_"`
	SearchBotService searchBotServiceClient.Config `env-prefix:"CHAT_SEARCH_SERVICE_"`
	Redis            repository.RedisConfig        `env-prefix:"REDIS_"`
	Postgres         postgres.Config               `env-prefix:"POSTGRES_"`
	UserService      userserviceclient.Config      `env-prefix:"USER_SERVICE_"`
}

// MustLoad читает конфигурацию из переменных окружения
func MustLoad() *Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("cannot read config from environment: " + err.Error())
	}

	return &cfg
}
