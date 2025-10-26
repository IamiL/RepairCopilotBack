package app

import (
	"fmt"
	"log/slog"
	"repairCopilotBot/tz-bot/internal/migrator"
	doctodocxconverterclient "repairCopilotBot/tz-bot/internal/pkg/docToDocxConverterClient"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	reportgeneratorclient "repairCopilotBot/tz-bot/internal/pkg/report-generator-client"
	telegramclient "repairCopilotBot/tz-bot/internal/pkg/telegram-client"
	user_service_client "repairCopilotBot/tz-bot/internal/pkg/user-service"
	"repairCopilotBot/tz-bot/internal/repository/postgres"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
	"time"

	grpcapp "repairCopilotBot/tz-bot/internal/app/grpc"
	tgapp "repairCopilotBot/tz-bot/internal/app/tg"
	"repairCopilotBot/tz-bot/internal/config"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	tzservice "repairCopilotBot/tz-bot/internal/service/tz"

	"github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	TokenTTL time.Duration `yaml:"token_ttl" env-default:"300h"`
	GRPCPort string        `yaml:"grpc_port" env-default:":50051"`
}

type App struct {
	GRPCServer  *grpcapp.App
	TelegramBot *tgapp.App
}

func New(
	log *slog.Logger,
	grpcConfig *grpcapp.Config,
	LlmConfig *tz_llm_client.Config,
	WordParserConfig *word_parser_client.Config,
	docToDocXConverterClientConfig *doctodocxconverterclient.Config,
	reportGeneratorClientConfig *reportgeneratorclient.Config,
	MarkdownServiceConfig *markdown_service_client.Config,
	PromtBuilderConfig *promt_builder.Config,
	s3Config *s3minio.Config,
	postgresConfig *postgres.Config,
	telegramBotConfig *config.TelegramBotConfig,
	telegramClientConfig *telegramclient.Config,
) *App {
	postgresConn, err := postgres.NewConnPool(postgresConfig)
	if err != nil {
		panic(err)
	}

	postgres, err := postgres.New(postgresConn)
	if err != nil {
		panic(err)
	}

	migratorRunner := migrator.NewMigrator(stdlib.OpenDB(*postgresConn.Config().ConnConfig.Copy()), postgresConfig.MigrationsDir)

	err = migratorRunner.Up()
	if err != nil {
		log.Error("Ошибка миграции базы данных: %v\n", err)
		panic(fmt.Errorf("cannot run migrator - %w", err).Error())
	}

	llmClient := tz_llm_client.NewWithCache(LlmConfig.Url, LlmConfig.Model, postgres)

	wordParserClient := word_parser_client.New(WordParserConfig.Url)

	//wordParserClient2 := word_parser2.NewWordConverterClient(WordParser2Config.Host, WordParser2Config.Port)

	docToDocXConverterClient := doctodocxconverterclient.NewClient(docToDocXConverterClientConfig.Host, docToDocXConverterClientConfig.Port)

	reportGeneratorClient := reportgeneratorclient.New(reportGeneratorClientConfig.Host, reportGeneratorClientConfig.Port)

	markdownClient := markdown_service_client.New(MarkdownServiceConfig.Url)

	prompBuilderClient := promt_builder.New(*PromtBuilderConfig)

	s3Conn, err := s3minio.NewConn(s3Config)
	if err != nil {
		panic(err)
	}

	s3Client := s3minio.New(s3Conn)

	// Создаем клиент для user-service (пока с фиксированным адресом, позже можно вынести в конфиг)
	userServiceClient, err := user_service_client.NewClient("localhost:8001")
	if err != nil {
		log.Error("failed to create user-service client", "error", err)
		// Если не удается подключиться к user-service, продолжаем без него
		userServiceClient = nil
	}

	// Создаем Telegram клиент для уведомлений об ошибках
	telegramClient, err := telegramclient.New(*telegramClientConfig)
	if err != nil {
		log.Error("failed to create telegram client", "error", err)
		// Если не удается создать telegram клиент, продолжаем без него
		telegramClient = nil
	}

	tzService := tzservice.New(log, wordParserClient, docToDocXConverterClient, reportGeneratorClient, markdownClient, llmClient, prompBuilderClient, userServiceClient, telegramClient, s3Client, postgres)

	grpcApp := grpcapp.New(log, tzService, grpcConfig)

	// Создаем Telegram бот
	telegramBot, err := tgapp.New(log, telegramBotConfig, tzService)
	if err != nil {
		log.Error("failed to create telegram bot", "error", err)
		// При ошибке создания бота продолжаем без него, но логируем ошибку
		telegramBot = nil
	}

	return &App{
		GRPCServer:  grpcApp,
		TelegramBot: telegramBot,
	}
}
