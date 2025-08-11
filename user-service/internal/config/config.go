package config

import (
	"flag"
	"os"
	grpcapp "repairCopilotBot/user-service/internal/app/grpc"
	"repairCopilotBot/user-service/internal/repository/postgres"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string          `yaml:"env" env-default:"local"`
	GRPC     grpcapp.Config  `yaml:"grpc_server"`
	Postgres postgres.Config `yaml:"postgres"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadPath(configPath)
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "user-service/config/local.yaml", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
