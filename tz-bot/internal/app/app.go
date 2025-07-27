package app

import (
	"log/slog"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"

	grpcapp "repairCopilotBot/tz-bot/internal/app/grpc"
	httpapp "repairCopilotBot/tz-bot/internal/app/http"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
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
	HTTPServer *httpapp.App
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger,
	appConfig *Config,
	httpConfig *httpapp.Config,
	grpcConfig *grpcapp.Config,
	LlmConfig *tz_llm_client.Config,
	WordParserConfig *word_parser_client.Config,
	TgConfig *tg_client.Config,
	s3Config *s3minio.Config,
) *App {
	llmClient := tz_llm_client.New(LlmConfig.Url)

	wordParserClient := word_parser_client.New(WordParserConfig.Url)

	tgBot, err := tg_client.NewBot(TgConfig.Token)
	if err != nil {
		panic(err)
	}

	tgClient := tg_client.New(tgBot, TgConfig.ChatID)

	s3Conn, err := s3minio.NewConn(s3Config)
	//if err != nil {
	//	panic(err)
	//}

	s3Client := s3minio.New(s3Conn)

	tzService := tzservice.New(log, wordParserClient, llmClient, tgClient, s3Client)

	httpApp := httpapp.New(log, httpConfig, tzService)

	grpcApp := grpcapp.New(log, tzService, grpcConfig)

	return &App{
		HTTPServer: httpApp,
		GRPCServer: grpcApp,
	}
}
