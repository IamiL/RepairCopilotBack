package tzservice

import (
	"context"
	"errors"
	"log/slog"
	doctodocxconverterclient "repairCopilotBot/tz-bot/internal/pkg/docToDocxConverterClient"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	telegramclient "repairCopilotBot/tz-bot/internal/pkg/telegram-client"
	user_service_client "repairCopilotBot/tz-bot/internal/pkg/user-service"
	word_parser_client "repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
	"sync"

	"github.com/google/uuid"
)

type Tz struct {
	log                      *slog.Logger
	wordConverterClient      *word_parser_client.Client
	docToDocXConverterClient *doctodocxconverterclient.Client
	reportGeneratorClient    ReportGeneratorClient
	markdownClient           *markdown_service_client.Client
	llmClient                *tz_llm_client.Client
	promtBuilderClient       *promt_builder.Client
	userServiceClient        *user_service_client.Client
	telegramClient           *telegramclient.Client
	s3                       *s3minio.MinioRepository
	repo                     Repository
	ggID                     int
	useLlmCache              bool
	mu                       sync.RWMutex
}

type ReportGeneratorClient interface {
	GenerateReport(ctx context.Context, payload any) ([]byte, error)
}

type ErrorSaver interface {
	SaveErrors(ctx context.Context, versionID uuid.UUID, errors *[]Error) error
	SaveInvalidInstances(ctx context.Context, invalidInstances *[]OutInvalidError) error
	SaveMissingInstances(ctx context.Context, missingInstances *[]OutMissingError) error
}

var (
	ErrConvertWordFile  = errors.New("error convert word file")
	ErrLlmAnalyzeFile   = errors.New("error in neural network file analysis")
	ErrGenerateDocxFile = errors.New("error in generate docx file")
)
