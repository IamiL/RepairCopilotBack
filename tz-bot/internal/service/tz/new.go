package tzservice

import (
	"log/slog"
	doctodocxconverterclient "repairCopilotBot/tz-bot/internal/pkg/docToDocxConverterClient"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	telegramclient "repairCopilotBot/tz-bot/internal/pkg/telegram-client"
	user_service_client "repairCopilotBot/tz-bot/internal/pkg/user-service"
	word_parser_client "repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
)

func New(
	log *slog.Logger,
	wordConverterClient *word_parser_client.Client,
	docToDocXConverterClient *doctodocxconverterclient.Client,
	reportGeneratorClient ReportGeneratorClient,
	markdownClient *markdown_service_client.Client,
	llmClient *tz_llm_client.Client,
	promtBuilder *promt_builder.Client,
	userServiceClient *user_service_client.Client,
	telegramClient *telegramclient.Client,
	s3 *s3minio.MinioRepository,
	repo Repository,
) *Tz {
	return &Tz{
		log:                      log,
		wordConverterClient:      wordConverterClient,
		docToDocXConverterClient: docToDocXConverterClient,
		reportGeneratorClient:    reportGeneratorClient,
		markdownClient:           markdownClient,
		llmClient:                llmClient,
		promtBuilderClient:       promtBuilder,
		userServiceClient:        userServiceClient,
		telegramClient:           telegramClient,
		s3:                       s3,
		repo:                     repo,
		ggID:                     6,
		useLlmCache:              true,
	}
}
