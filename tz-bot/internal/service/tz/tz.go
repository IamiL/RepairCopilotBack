package tzservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
)

type Tz struct {
	log                 *slog.Logger
	wordConverterClient *word_parser_client.Client
	markdownClient      *markdown_service_client.Client
	llmClient           *tz_llm_client.Client
	tgClient            *tg_client.Client
	s3                  *s3minio.MinioRepository
	repo                repository.Repository
}

var (
	ErrConvertWordFile  = errors.New("error convert word file")
	ErrLlmAnalyzeFile   = errors.New("error in neural network file analysis")
	ErrGenerateDocxFile = errors.New("error in generate docx file")
)

type TzError struct {
	Id    string
	Title string
	Text  string
	Type  string
}

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

func New(
	log *slog.Logger,
	wordConverterClient *word_parser_client.Client,
	markdownClient *markdown_service_client.Client,
	llmClient *tz_llm_client.Client,
	tgClient *tg_client.Client,
	s3 *s3minio.MinioRepository,
	repo repository.Repository,
) *Tz {
	return &Tz{
		log:                 log,
		wordConverterClient: wordConverterClient,
		markdownClient:      markdownClient,
		llmClient:           llmClient,
		tgClient:            tgClient,
		s3:                  s3,
		repo:                repo,
	}
}

func (tz *Tz) CheckTz(ctx context.Context, file []byte, filename string, userID uuid.UUID) (string, string, string, *[]OutInvalidError, *[]OutMissingError, string, error) {
	const op = "Tz.CheckTz"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("checking tz")

	htmlText, css, err := tz.wordConverterClient.Convert(file, filename)
	if err != nil {
		tz.log.Info("Ошибка обработки файла в wordConverterClient: %v\n" + err.Error())

		//tz.tgClient.SendMessage("Ошибка обработки файла в wordConverterClient: %v\n" + err.Error())

		return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", ErrConvertWordFile
	}

	log.Info("конвертация word файла в htmlText успешна")

	log.Info("отправляем HTML в markdown-service для конвертации")

	markdownResponse, err := tz.markdownClient.Convert(*htmlText)
	if err != nil {
		log.Error("ошибка конвертации HTML в markdown: ", sl.Err(err))
		//tz.tgClient.SendMessage(fmt.Sprintf("Ошибка конвертации HTML в markdown: %v", err))
		return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", fmt.Errorf("ошибка конвертации HTML в markdown: %w", err)
	}

	log.Info("конвертация HTML в markdown успешна")
	log.Info(fmt.Sprintf("получены дополнительные данные: message=%s, mappings_count=%d", markdownResponse.Message, len(markdownResponse.Mappings)))

	//log.Info("отправка Markdown файла в телеграм")
	//
	//markdownFileName := strings.TrimSuffix(filename, ".docx") + ".md"
	//markdownFileData := []byte(markdownResponse.Markdown)
	//err = tz.tgClient.SendFile(markdownFileData, markdownFileName)
	//if err != nil {
	//	log.Error("ошибка отправки Markdown файла в телеграм: ", sl.Err(err))
	//	//tz.tgClient.SendMessage(fmt.Sprintf("Ошибка отправки Markdown файла в телеграм: %v", err))
	//} else {
	//	log.Info("Markdown файл успешно отправлен в телеграм")
	//}

	llmAnalyzeResult, err := tz.llmClient.Analyze(markdownResponse.Markdown)
	if err != nil {
		log.Error("Error: \n", err)
	}
	if llmAnalyzeResult == nil {
		//tz.tgClient.SendMessage("ИСПРАВИТЬ: от llm пришёл пустой ответ, но код ответа не ошибочный.")

		log.Info("пустой ответ от llm")
		return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", ErrLlmAnalyzeFile
	}
	if llmAnalyzeResult.Reports == nil || len(llmAnalyzeResult.Reports) == 0 {
		//tz.tgClient.SendMessage("МБ ЧТО-ТО НЕ ТАК: от llm ответ без отчетов, но код ответа не ошибочный")

		log.Info("0 отчетов в ответе от llm")
		return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", ErrLlmAnalyzeFile
	}

	outInvalidErrors, outMissingErrors, outHtml := HandleErrors(&llmAnalyzeResult.Reports, &markdownResponse.Mappings)

	LogOutInvalidErrors(log, outInvalidErrors, "После сортировки")

	// Сохраняем оригинальный файл в S3
	originalFileID := uuid.New().String()
	err = tz.s3.SaveDocument(ctx, originalFileID, file)
	if err != nil {
		log.Error("ошибка сохранения оригинального файла в S3: ", sl.Err(err))
		return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", fmt.Errorf("ошибка сохранения файла в S3: %w", err)
	}

	log.Info("оригинальный файл успешно сохранён в S3", slog.String("file_id", originalFileID))

	// Сохраняем данные в БД
	err = tz.saveTechnicalSpecificationData(ctx, filename, userID, outHtml, *css, originalFileID, outInvalidErrors, outMissingErrors, log)
	if err != nil {
		log.Error("ошибка сохранения данных в БД: ", sl.Err(err))
		// Не возвращаем ошибку, чтобы не блокировать ответ пользователю
	}

	return outHtml, *css, "123", outInvalidErrors, outMissingErrors, "123", nil
}

// saveTechnicalSpecificationData saves technical specification data to database
func (tz *Tz) saveTechnicalSpecificationData(
	ctx context.Context,
	filename string,
	userID uuid.UUID,
	outHTML string,
	css string,
	originalFileID string,
	invalidErrors *[]OutInvalidError,
	missingErrors *[]OutMissingError,
	log *slog.Logger,
) error {
	now := time.Now()

	// Создаем техническое задание
	tsID := uuid.New()
	tsReq := &repository.CreateTechnicalSpecificationRequest{
		ID:        tsID,
		Name:      filename,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ts, err := tz.repo.CreateTechnicalSpecification(ctx, tsReq)
	if err != nil {
		return fmt.Errorf("failed to create technical specification: %w", err)
	}

	log.Info("technical specification created", slog.String("ts_id", ts.ID.String()))

	// Создаем версию
	versionID := uuid.New()
	versionReq := &repository.CreateVersionRequest{
		ID:                       versionID,
		TechnicalSpecificationID: tsID,
		VersionNumber:            1, // Первая версия
		CreatedAt:                now,
		UpdatedAt:                now,
		OriginalFileID:           originalFileID,
		OutHTML:                  outHTML,
		CSS:                      css,
		CheckedFileID:            "", // Пока пустое
	}

	version, err := tz.repo.CreateVersion(ctx, versionReq)
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}

	log.Info("version created", slog.String("version_id", version.ID.String()))

	// Сохраняем InvalidErrors
	if invalidErrors != nil && len(*invalidErrors) > 0 {
		invalidErrorData := make([]repository.InvalidErrorData, 0, len(*invalidErrors))
		for _, err := range *invalidErrors {
			invalidErrorData = append(invalidErrorData, repository.InvalidErrorData{
				ID:           uuid.New(),
				ErrorID:      int(err.Id),
				ErrorIDStr:   err.IdStr,
				GroupID:      err.GroupID,
				ErrorCode:    err.ErrorCode,
				Quote:        err.Quote,
				Analysis:     err.Analysis,
				Critique:     err.Critique,
				Verification: err.Verification,
				SuggestedFix: err.SuggestedFix,
				Rationale:    err.Rationale,
				CreatedAt:    now,
			})
		}

		invalidReq := &repository.CreateInvalidErrorsRequest{
			VersionID: versionID,
			Errors:    invalidErrorData,
		}

		err = tz.repo.CreateInvalidErrors(ctx, invalidReq)
		if err != nil {
			return fmt.Errorf("failed to create invalid errors: %w", err)
		}

		log.Info("invalid errors saved", slog.Int("count", len(invalidErrorData)))
	}

	// Сохраняем MissingErrors
	if missingErrors != nil && len(*missingErrors) > 0 {
		missingErrorData := make([]repository.MissingErrorData, 0, len(*missingErrors))
		for _, err := range *missingErrors {
			missingErrorData = append(missingErrorData, repository.MissingErrorData{
				ID:           uuid.New(),
				ErrorID:      int(err.Id),
				ErrorIDStr:   err.IdStr,
				GroupID:      err.GroupID,
				ErrorCode:    err.ErrorCode,
				Analysis:     err.Analysis,
				Critique:     err.Critique,
				Verification: err.Verification,
				SuggestedFix: err.SuggestedFix,
				Rationale:    err.Rationale,
				CreatedAt:    now,
			})
		}

		missingReq := &repository.CreateMissingErrorsRequest{
			VersionID: versionID,
			Errors:    missingErrorData,
		}

		err = tz.repo.CreateMissingErrors(ctx, missingReq)
		if err != nil {
			return fmt.Errorf("failed to create missing errors: %w", err)
		}

		log.Info("missing errors saved", slog.Int("count", len(missingErrorData)))
	}

	log.Info("technical specification data saved successfully")
	return nil
}

func (tz *Tz) GetTechnicalSpecificationVersions(ctx context.Context, userID uuid.UUID) ([]*repository.VersionSummary, error) {
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

func (tz *Tz) GetVersion(ctx context.Context, versionID uuid.UUID) (string, string, string, *[]OutInvalidError, *[]OutMissingError, string, error) {
	const op = "Tz.GetVersion"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("versionID", versionID.String()),
	)

	log.Info("getting version with errors")

	version, invalidErrors, missingErrors, err := tz.repo.GetVersionWithErrors(ctx, versionID)
	if err != nil {
		log.Error("failed to get version with errors: ", sl.Err(err))
		return "", "", "", nil, nil, "", fmt.Errorf("failed to get version with errors: %w", err)
	}

	// Конвертируем InvalidError в OutInvalidError
	outInvalidErrors := make([]OutInvalidError, len(invalidErrors))
	for i, invErr := range invalidErrors {
		outInvalidErrors[i] = OutInvalidError{
			Id:           uint32(invErr.ErrorID),
			IdStr:        invErr.ErrorIDStr,
			GroupID:      invErr.GroupID,
			ErrorCode:    invErr.ErrorCode,
			Quote:        invErr.Quote,
			Analysis:     invErr.Analysis,
			Critique:     invErr.Critique,
			Verification: invErr.Verification,
			SuggestedFix: invErr.SuggestedFix,
			Rationale:    invErr.Rationale,
		}
	}

	// Конвертируем MissingError в OutMissingError
	outMissingErrors := make([]OutMissingError, len(missingErrors))
	for i, missErr := range missingErrors {
		outMissingErrors[i] = OutMissingError{
			Id:           uint32(missErr.ErrorID),
			IdStr:        missErr.ErrorIDStr,
			GroupID:      missErr.GroupID,
			ErrorCode:    missErr.ErrorCode,
			Analysis:     missErr.Analysis,
			Critique:     missErr.Critique,
			Verification: missErr.Verification,
			SuggestedFix: missErr.SuggestedFix,
			Rationale:    missErr.Rationale,
		}
	}

	log.Info("version with errors retrieved successfully", 
		slog.Int("invalid_errors_count", len(outInvalidErrors)),
		slog.Int("missing_errors_count", len(outMissingErrors)))
	
	return version.OutHTML, version.CSS, version.CheckedFileID, &outInvalidErrors, &outMissingErrors, version.CheckedFileID, nil
}
