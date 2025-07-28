package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"repairCopilotBot/api-gateway-service/internal/app"
	httpapp "repairCopilotBot/api-gateway-service/internal/app/http"
	"repairCopilotBot/api-gateway-service/internal/pkg/tg"
	"repairCopilotBot/tz-bot/client"
)

type Config struct {
	Env          string           `yaml:"env" env-default:"local"`
	App          app.Config       `yaml:"app"`
	HTTP         httpapp.Config   `yaml:"http_server"`
	Tg           tg_client.Config `yaml:"tg_client"`
	TzBotService client.Config    `yaml:"tz_bot_service"`
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

	flag.StringVar(&res, "config", "api-gateway-service/config/config.yaml", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
