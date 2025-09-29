package tzservice

import (
	"log/slog"
	doctodocxconverterclient "repairCopilotBot/tz-bot/internal/pkg/docToDocxConverterClient"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	user_service_client "repairCopilotBot/tz-bot/internal/pkg/user-service"
	word_parser_client "repairCopilotBot/tz-bot/internal/pkg/word-parser"
	word_parser2 "repairCopilotBot/tz-bot/internal/pkg/word-parser2"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
)

func New(
	log *slog.Logger,
	wordConverterClient *word_parser_client.Client,
	wordConverterClient2 *word_parser2.WordConverterClient,
	docToDocXConverterClient *doctodocxconverterclient.Client,
	reportGeneratorClient ReportGeneratorClient,
	markdownClient *markdown_service_client.Client,
	llmClient *tz_llm_client.Client,
	promtBuilder *promt_builder.Client,
	userServiceClient *user_service_client.Client,
	s3 *s3minio.MinioRepository,
	repo Repository,
) *Tz {
	return &Tz{
		log:                      log,
		wordConverterClient:      wordConverterClient,
		wordConverterClient2:     wordConverterClient2,
		docToDocXConverterClient: docToDocXConverterClient,
		reportGeneratorClient:    reportGeneratorClient,
		markdownClient:           markdownClient,
		llmClient:                llmClient,
		promtBuilderClient:       promtBuilder,
		userServiceClient:        userServiceClient,
		s3:                       s3,
		repo:                     repo,
		ggID:                     6,
		useLlmCache:              true,
	}
}
