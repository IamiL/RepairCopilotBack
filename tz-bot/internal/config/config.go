package config

import (
	"flag"
	"os"
	"repairCopilotBot/tz-bot/internal/app"
	grpcapp "repairCopilotBot/tz-bot/internal/app/grpc"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	word_parser2 "repairCopilotBot/tz-bot/internal/pkg/word-parser2"
	"repairCopilotBot/tz-bot/internal/repository/postgres"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             string                         `yaml:"env" env-default:"local"`
	App             app.Config                     `yaml:"app"`
	GRPC            grpcapp.Config                 `yaml:"grpc_server"`
	Llm             tz_llm_client.Config           `yaml:"llm_client"`
	PromtBuilder    promt_builder.Config           `yaml:"promt_builder"`
	Tg              tg_client.Config               `yaml:"tg_client"`
	WordParser      word_parser_client.Config      `yaml:"word_parser_client"`
	WordParser2     word_parser2.Config            `yaml:"word_parser_client2"`
	MarkdownService markdown_service_client.Config `yaml:"markdown_service_client"`
	S3minio         s3minio.Config                 `yaml:"s3minio"`
	Postgres        postgres.Config                `yaml:"postgres"`
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
