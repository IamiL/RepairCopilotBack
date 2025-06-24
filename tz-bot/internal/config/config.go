package config

import (
	"flag"
	"os"
	"repairCopilotBot/tz-bot/internal/app"
	httpapp "repairCopilotBot/tz-bot/internal/app/http"
	tz_llm_client "repairCopilotBot/tz-bot/package/llm"
	tg_client "repairCopilotBot/tz-bot/package/tg"
	word_parser_client "repairCopilotBot/tz-bot/package/word-parser"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string                    `yaml:"env" env-default:"local"`
	App        app.Config                `yaml:"app"`
	HTTP       httpapp.Config            `yaml:"http_server"`
	Llm        tz_llm_client.Config      `yaml:"llm_client"`
	Tg         tg_client.Config          `yaml:"tg_client"`
	WordParser word_parser_client.Config `yaml:"word_parser_client"`
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

	flag.StringVar(&res, "config", "tz-bot/config/config.yaml", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
