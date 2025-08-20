package tzservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	modelrepo "repairCopilotBot/tz-bot/internal/repository/models"
	"sort"
	"strconv"

	"github.com/google/uuid"

	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
)

type Tz struct {
	log                 *slog.Logger
	wordConverterClient *word_parser_client.Client
	markdownClient      *markdown_service_client.Client
	llmClient           *tz_llm_client.Client
	promtBuilderClient  *promt_builder.Client
	tgClient            *tg_client.Client
	s3                  *s3minio.MinioRepository
	repo                Repository
	ggID                int
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

// ErrorInstance описывает одну ошибку из LLM API
type ErrorInstance struct {
	GroupID      string  `json:"group_id"`
	Code         string  `json:"code"`
	ErrType      string  `json:"err_type"`
	Snippet      string  `json:"snippet"`
	LineStart    *int    `json:"line_start"`
	LineEnd      *int    `json:"line_end"`
	SuggestedFix *string `json:"suggested_fix"`
	Rationale    string  `json:"rationale"`
}

// HtmlBlock связывает HTML-блок с Markdown-диапазоном
type HtmlBlock struct {
	ElementID     string `json:"html_element_id"`
	HtmlContent   string `json:"html_content"`
	MarkdownStart int    `json:"markdown_line_start"`
	MarkdownEnd   int    `json:"markdown_line_end"`
}

type OutError struct {
	ID           string  `json:"id"`
	GroupID      string  `json:"group_id"`
	Code         string  `json:"code"`
	SuggestedFix *string `json:"suggested_fix"`
	Rationale    string  `json:"rationale"`
}

// llmRequestResult represents the result of a single LLM request
type llmRequestResult struct {
	groupReport *tz_llm_client.GroupReport
	cost        *float64
	tokens      *int64
	err         error
}

func New(
	log *slog.Logger,
	wordConverterClient *word_parser_client.Client,
	markdownClient *markdown_service_client.Client,
	llmClient *tz_llm_client.Client,
	promtBuilder *promt_builder.Client,
	tgClient *tg_client.Client,
	s3 *s3minio.MinioRepository,
	repo Repository,
) *Tz {
	return &Tz{
		log:                 log,
		wordConverterClient: wordConverterClient,
		markdownClient:      markdownClient,
		llmClient:           llmClient,
		promtBuilderClient:  promtBuilder,
		tgClient:            tgClient,
		s3:                  s3,
		repo:                repo,
		ggID:                1,
	}
}

// saveTechnicalSpecificationData saves technical specification data to database
//func (tz *Tz) saveTechnicalSpecificationData(
//	ctx context.Context,
//	filename string,
//	userID uuid.UUID,
//	outHTML string,
//	css string,
//	originalFileID string,
//	invalidErrors *[]OutInvalidError,
//	missingErrors *[]OutMissingError,
//	errors *[]Error,
//	allRubs float64,
//	allTokens int64,
//	inspectionTime time.Duration,
//	log *slog.Logger,
//) error {
//	now := time.Now()
//
//	// Создаем техническое задание
//	tsID := uuid.New()
//	tsReq := &modelrepo.CreateTechnicalSpecificationRequest{
//		ID:        tsID,
//		Name:      filename,
//		UserID:    userID,
//		CreatedAt: now,
//		UpdatedAt: now,
//	}
//
//	ts, err := tz.repo.CreateTechnicalSpecification(ctx, tsReq)
//	if err != nil {
//		return fmt.Errorf("failed to create technical specification: %w", err)
//	}
//
//	log.Info("technical specification created", slog.String("ts_id", ts.ID.String()))
//
//	// Создаем версию
//	versionID := uuid.New()
//	versionReq := &repository.CreateVersionRequest{
//		ID:                       versionID,
//		TechnicalSpecificationID: tsID,
//		VersionNumber:            1, // Первая версия
//		CreatedAt:                now,
//		UpdatedAt:                now,
//		OriginalFileID:           originalFileID,
//		OutHTML:                  outHTML,
//		CSS:                      css,
//		CheckedFileID:            "", // Пока пустое
//		AllRubs:                  allRubs,
//		AllTokens:                allTokens,
//		InspectionTime:           inspectionTime,
//	}
//
//	version, err := tz.repo.CreateVersion(ctx, versionReq)
//	if err != nil {
//		return fmt.Errorf("failed to create version: %w", err)
//	}
//
//	log.Info("version created", slog.String("version_id", version.ID.String()))
//
//	// Сохраняем InvalidErrors
//	if invalidErrors != nil && len(*invalidErrors) > 0 {
//		invalidErrorData := make([]repository.InvalidErrorData, 0, len(*invalidErrors))
//		for i, err := range *invalidErrors {
//			invalidErrorData = append(invalidErrorData, repository.InvalidErrorData{
//				ID:           uuid.New(),
//				ErrorID:      int(err.Id),
//				ErrorIDStr:   err.IdStr,
//				GroupID:      err.GroupID,
//				ErrorCode:    err.ErrorCode,
//				Quote:        err.Quote,
//				Analysis:     err.Analysis,
//				Critique:     err.Critique,
//				Verification: err.Verification,
//				SuggestedFix: err.SuggestedFix,
//				Rationale:    err.Rationale,
//				OrderNumber:  i, // Порядковый номер (индекс в массиве)
//				CreatedAt:    now,
//			})
//		}
//
//		invalidReq := &repository.CreateInvalidErrorsRequest{
//			VersionID: versionID,
//			Errors:    invalidErrorData,
//		}
//
//		err = tz.repo.CreateInvalidErrors(ctx, invalidReq)
//		if err != nil {
//			return fmt.Errorf("failed to create invalid errors: %w", err)
//		}
//
//		log.Info("invalid errors saved", slog.Int("count", len(invalidErrorData)))
//	}
//
//	// Сохраняем MissingErrors
//	if missingErrors != nil && len(*missingErrors) > 0 {
//		missingErrorData := make([]repository.MissingErrorData, 0, len(*missingErrors))
//		for _, err := range *missingErrors {
//			missingErrorData = append(missingErrorData, repository.MissingErrorData{
//				ID:           uuid.New(),
//				ErrorID:      int(err.Id),
//				ErrorIDStr:   err.IdStr,
//				GroupID:      err.GroupID,
//				ErrorCode:    err.ErrorCode,
//				Analysis:     err.Analysis,
//				Critique:     err.Critique,
//				Verification: err.Verification,
//				SuggestedFix: err.SuggestedFix,
//				Rationale:    err.Rationale,
//				CreatedAt:    now,
//			})
//		}
//
//		missingReq := &repository.CreateMissingErrorsRequest{
//			VersionID: versionID,
//			Errors:    missingErrorData,
//		}
//
//		err = tz.repo.CreateMissingErrors(ctx, missingReq)
//		if err != nil {
//			return fmt.Errorf("failed to create missing errors: %w", err)
//		}
//
//		log.Info("missing errors saved", slog.Int("count", len(missingErrorData)))
//	}
//
//	// Сохраняем Errors
//	if errors != nil && len(*errors) > 0 {
//		errorData := make([]repository.ErrorData, 0, len(*errors))
//		for _, err := range *errors {
//			instancesJSON, jsonErr := json.Marshal(err.Instances)
//			if jsonErr != nil {
//				return fmt.Errorf("failed to marshal instances: %w", jsonErr)
//			}
//
//			errorData = append(errorData, repository.ErrorData{
//				ID:                  err.ID,
//				GroupID:             &err.GroupID,
//				ErrorCode:           &err.ErrorCode,
//				PreliminaryNotes:    err.PreliminaryNotes,
//				OverallCritique:     err.OverallCritique,
//				Verdict:             &err.Verdict,
//				ProcessAnalysis:     err.ProcessAnalysis,
//				ProcessCritique:     err.ProcessCritique,
//				ProcessVerification: err.ProcessVerification,
//				ProcessRetrieval:    err.ProcessRetrieval,
//				Instances:           instancesJSON,
//			})
//		}
//
//		errorsReq := &repository.CreateErrorsRequest{
//			VersionID: versionID,
//			Errors:    errorData,
//		}
//
//		err = tz.repo.CreateErrors(ctx, errorsReq)
//		if err != nil {
//			return fmt.Errorf("failed to create errors: %w", err)
//		}
//
//		log.Info("errors saved", slog.Int("count", len(errorData)))
//	}
//
//	log.Info("technical specification data saved successfully")
//	return nil
//}

func (tz *Tz) GetTechnicalSpecificationVersions(ctx context.Context, userID uuid.UUID) ([]*modelrepo.VersionSummary, error) {
	const op = "Tz.GetTechnicalSpecificationVersions"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("getting technical specification versions")

	versions, err := tz.repo.GetVersionsByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get versions by user ID: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get versions by user ID: %w", err)
	}

	log.Info("technical specification versions retrieved successfully", slog.Int("count", len(versions)))
	return versions, nil
}

func (tz *Tz) GetAllVersions(ctx context.Context) ([]*modelrepo.VersionWithErrorCounts, error) {
	const op = "Tz.GetAllVersions"

	log := tz.log.With(
		slog.String("op", op),
	)

	log.Info("getting all versions with error counts")

	versions, err := tz.repo.GetAllVersions(ctx)
	if err != nil {
		log.Error("failed to get all versions: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get all versions: %w", err)
	}

	log.Info("all versions retrieved successfully", slog.Int("count", len(versions)))
	return versions, nil
}

func (tz *Tz) GetVersionStatistics(ctx context.Context) (*modelrepo.VersionStatistics, error) {
	const op = "Tz.GetVersionStatistics"

	log := tz.log.With(
		slog.String("op", op),
	)

	log.Info("getting version statistics")

	stats, err := tz.repo.GetVersionStatistics(ctx)
	if err != nil {
		log.Error("failed to get version statistics: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get version statistics: %w", err)
	}

	log.Info("version statistics retrieved successfully",
		slog.Int64("total_versions", stats.TotalVersions),
		slog.Any("total_tokens", stats.TotalTokens),
		slog.Any("total_rubs", stats.TotalRubs),
		slog.Any("average_inspection_time", stats.AverageInspectionTime))

	return stats, nil
}

func (tz *Tz) GetVersion(ctx context.Context, versionID uuid.UUID) (string, string, string, *[]Error, *[]OutInvalidError, string, error) {
	const op = "Tz.GetVersion"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("versionID", versionID.String()),
	)

	log.Info("getting version with errors")

	version, err := tz.repo.GetVersion(ctx, versionID)
	if err != nil {
		log.Error("failed to get version: ", sl.Err(err))
		return "", "", "", nil, nil, "", err
	}

	errorsInTz, err := tz.repo.GetErrorsByVersionID(ctx, versionID)
	if err != nil {
		log.Error("failed to get version errors: ", sl.Err(err))
		return "", "", "", nil, nil, "", err
	}

	invalidInstances := make([]OutInvalidError, 0)
	for i := range *errorsInTz {
		invalidInstancesFromDb, err := tz.repo.GetInvalidInstancesByErrorID(ctx, (*errorsInTz)[i].ID)
		if err != nil {
			log.Error("failed to get version errors: ", sl.Err(err))
		} else {
			for j := range *invalidInstancesFromDb {
				(*invalidInstancesFromDb)[j].HtmlIDStr = strconv.Itoa(int((*invalidInstancesFromDb)[j].HtmlID))
			}
			invalidInstances = append(invalidInstances, *invalidInstancesFromDb...)
			(*errorsInTz)[i].InvalidInstances = &invalidInstances
		}

		missingInstances, err := tz.repo.GetMissingInstancesByErrorID(ctx, (*errorsInTz)[i].ID)
		if err != nil {
			log.Error("failed to get version missing instances: ", sl.Err(err))
		} else {
			(*errorsInTz)[i].MissingInstances = missingInstances
		}
	}

	SortOutInvalidErrorsByOrderNumber(&invalidInstances)

	//version, invalidErrors, missingErrors, err := tz.repo.GetVersionWithErrors(ctx, versionID)
	//if err != nil {
	//	log.Error("failed to get version with errors: ", sl.Err(err))
	//	return "", "", "", nil, nil, "", fmt.Errorf("failed to get version with errors: %w", err)
	//}
	//
	//// Конвертируем InvalidError в OutInvalidError
	//outInvalidErrors := make([]OutInvalidError, len(invalidErrors))
	//for i, invErr := range invalidErrors {
	//	outInvalidErrors[i] = OutInvalidError{
	//		Id:           uint32(invErr.ErrorID),
	//		IdStr:        invErr.ErrorIDStr,
	//		GroupID:      invErr.GroupID,
	//		ErrorCode:    invErr.ErrorCode,
	//		Quote:        invErr.Quote,
	//		Analysis:     invErr.Analysis,
	//		Critique:     invErr.Critique,
	//		Verification: invErr.Verification,
	//		SuggestedFix: invErr.SuggestedFix,
	//		Rationale:    invErr.Rationale,
	//	}
	//}
	//
	//// Конвертируем MissingError в OutMissingError
	//outMissingErrors := make([]OutMissingError, len(missingErrors))
	//for i, missErr := range missingErrors {
	//	outMissingErrors[i] = OutMissingError{
	//		Id:           uint32(missErr.ErrorID),
	//		IdStr:        missErr.ErrorIDStr,
	//		GroupID:      missErr.GroupID,
	//		ErrorCode:    missErr.ErrorCode,
	//		Analysis:     missErr.Analysis,
	//		Critique:     missErr.Critique,
	//		Verification: missErr.Verification,
	//		SuggestedFix: missErr.SuggestedFix,
	//		Rationale:    missErr.Rationale,
	//	}
	//}

	//log.Info("version with errors retrieved successfully",
	//	slog.Int("invalid_errors_count", len(outInvalidErrors)),
	//	slog.Int("missing_errors_count", len(outMissingErrors)))

	return version.OutHTML, version.CSS, version.CheckedFileID, errorsInTz, &invalidInstances, "", nil
}

func SortOutInvalidErrorsByOrderNumber(errors *[]OutInvalidError) {
	if errors == nil {
		return
	}

	sort.Slice(*errors, func(i, j int) bool {
		return (*errors)[i].OrderNumber < (*errors)[j].OrderNumber
	})
}

func (tz *Tz) NewFeedbackError(ctx context.Context, versionID uuid.UUID, errorID, errorType string, feedbackType uint32, comment string, userID uuid.UUID) error {
	const op = "Tz.NewFeedbackError"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("versionID", versionID.String()),
		slog.String("errorID", errorID),
		slog.String("errorType", errorType),
		slog.String("userID", userID.String()),
	)

	log.Info("creating feedback error")

	// Парсим errorID в int и получаем UUID через репозиторий
	//errorIDInt, err := strconv.Atoi(errorID)
	//if err != nil {
	//	log.Error("invalid error ID format", slog.String("error", err.Error()))
	//	return fmt.Errorf("invalid error ID format: %w", err)
	//}

	//errorUUID, err := tz.repo.GetUUIDByErrorID(ctx, errorIDInt)
	//if err != nil {
	//	log.Error("failed to get UUID by error ID", slog.String("error", err.Error()))
	//	return fmt.Errorf("failed to get UUID by error ID: %w", err)
	//}

	// Определяем тип ошибки
	//var repoErrorType modelrepo.ErrorType
	//switch errorType {
	//case "invalid":
	//	repoErrorType = modelrepo.ErrorTypeInvalid
	//case "missing":
	//	repoErrorType = modelrepo.ErrorTypeMissing
	//default:
	//	log.Error("invalid error type", slog.String("errorType", errorType))
	//	return fmt.Errorf("invalid error type: %s", errorType)
	//}

	//now := time.Now()

	// Создаем запрос для создания обратной связи об ошибке
	//feedbackReq := &repository.CreateErrorFeedbackRequest{
	//	ID:           uuid.New(),
	//	VersionID:    versionID,
	//	ErrorID:      errorUUID,
	//	ErrorType:    repoErrorType,
	//	UserID:       userID,
	//	FeedbackType: int(feedbackType),
	//	Comment:      &comment,
	//	CreatedAt:    now,
	//	UpdatedAt:    now,
	//}
	//
	//// Если комментарий пустой, устанавливаем nil
	//if comment == "" {
	//	feedbackReq.Comment = nil
	//}

	// Сохраняем обратную связь в БД
	//_, err = tz.repo.CreateErrorFeedback(ctx, feedbackReq)
	//if err != nil {
	//	log.Error("failed to create error feedback", slog.String("error", err.Error()))
	//	return fmt.Errorf("failed to create error feedback: %w", err)
	//}

	log.Info("feedback error created successfully")
	return nil
}
