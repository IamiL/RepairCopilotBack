package config

import (
	grpcapp "repairCopilotBot/tz-bot/internal/app/grpc"
	doctodocxconverterclient "repairCopilotBot/tz-bot/internal/pkg/docToDocxConverterClient"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	reportgeneratorclient "repairCopilotBot/tz-bot/internal/pkg/report-generator-client"
	telegramclient "repairCopilotBot/tz-bot/internal/pkg/telegram-client"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	word_parser2 "repairCopilotBot/tz-bot/internal/pkg/word-parser2"
	"repairCopilotBot/tz-bot/internal/repository/postgres"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env                      string                          `env:"ENV" env-default:"local"`
	GRPC                     grpcapp.Config                  `env-prefix:"GRPC_"`
	Llm                      tz_llm_client.Config            `env-prefix:"LLM_"`
	PromtBuilder             promt_builder.Config            `env-prefix:"PROMT_BUILDER_"`
	WordParser               word_parser_client.Config       `env-prefix:"DOCX_PARSER_"`
	WordParser2              word_parser2.Config             `env-prefix:"WORD_PARSER_"`
	DocToDocXConverterClient doctodocxconverterclient.Config `env-prefix:"DOC_TO_DOCX_CONVERTER_"`
	ReportGeneratorClient    reportgeneratorclient.Config    `env-prefix:"REPORT_GENERATOR_"`
	MarkdownService          markdown_service_client.Config  `env-prefix:"MD_CONVERTER_"`
	S3minio                  s3minio.Config                  `env-prefix:"S3_MINIO_"`
	Postgres                 postgres.Config                 `env-prefix:"POSTGRES_"`
	TelegramBot              TelegramBotConfig               `env-prefix:"TELEGRAM_BOT_"`
	TelegramClient           telegramclient.Config           `env-prefix:"TELEGRAM_CLIENT_"`
}

type TelegramBotConfig struct {
	Token       string `env:"TOKEN" env-required:"true"`
	ChatID      string `env:"CHAT_ID" env-required:"true"`
	UseWebhooks bool   `env:"USE_WEBHOOKS" env-default:"false"`
	WebhookHost string `env:"WEBHOOK_HOST" env-default:""`
	WebhookPort int    `env:"WEBHOOK_PORT" env-default:"8443"`
	WebhookPath string `env:"WEBHOOK_PATH" env-default:"/webhook"`
}

// MustLoad читает конфигурацию из переменных окружения
func MustLoad() *Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("cannot read config from environment: " + err.Error())
	}

	return &cfg
}
