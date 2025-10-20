package config

import (
	grpcapp "repairCopilotBot/user-service/internal/app/grpc"
	"repairCopilotBot/user-service/internal/repository/postgres"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string          `env:"ENV" env-default:"local"`
	GRPC     grpcapp.Config  `env-prefix:"GRPC_"`
	Postgres postgres.Config `env-prefix:"POSTGRES_"`
}

// MustLoad читает конфигурацию из переменных окружения
func MustLoad() *Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("cannot read config from environment: " + err.Error())
	}

	return &cfg
}
