package app

import (
	"log/slog"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	word_parser2 "repairCopilotBot/tz-bot/internal/pkg/word-parser2"
	"repairCopilotBot/tz-bot/internal/repository/postgres"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"

	grpcapp "repairCopilotBot/tz-bot/internal/app/grpc"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	tzservice "repairCopilotBot/tz-bot/internal/service/tz"
	"time"
)

type Config struct {
	TokenTTL time.Duration `yaml:"token_ttl" env-default:"300h"`
	GRPCPort string        `yaml:"grpc_port" env-default:":50051"`
}

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcConfig *grpcapp.Config,
	LlmConfig *tz_llm_client.Config,
	WordParserConfig *word_parser_client.Config,
	WordParser2Config *word_parser2.Config,
	MarkdownServiceConfig *markdown_service_client.Config,
	PromtBuilderConfig *promt_builder.Config,
	TgConfig *tg_client.Config,
	s3Config *s3minio.Config,
	postgresConfig *postgres.Config,
) *App {
	postgresConn, err := postgres.NewConnPool(postgresConfig)
	if err != nil {
		panic(err)
	}

	postgres, err := postgres.New(postgresConn)
	if err != nil {
		panic(err)
	}

	llmClient := tz_llm_client.NewWithCache(LlmConfig.Url, LlmConfig.Model, postgres)

	wordParserClient := word_parser_client.New(WordParserConfig.Url)

	wordParserClient2 := word_parser2.NewWordConverterClient(WordParser2Config.Host, WordParser2Config.Port)

	markdownClient := markdown_service_client.New(MarkdownServiceConfig.Url)

	prompBuilderClient := promt_builder.New(PromtBuilderConfig.Url)

	tgBot, err := tg_client.NewBot(TgConfig.Token)
	if err != nil {
		panic(err)
	}

	tgClient := tg_client.New(tgBot, TgConfig.ChatID)

	s3Conn, err := s3minio.NewConn(s3Config)
	if err != nil {
		panic(err)
	}

	s3Client := s3minio.New(s3Conn)

	tzService := tzservice.New(log, wordParserClient, wordParserClient2, markdownClient, llmClient, prompBuilderClient, tgClient, s3Client, postgres)

	grpcApp := grpcapp.New(log, tzService, grpcConfig)

	return &App{
		GRPCServer: grpcApp,
	}
}
