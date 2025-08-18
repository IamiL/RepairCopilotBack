package tzservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	"strconv"
	"sync"
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
	promtBuilderClient  *promt_builder.Client
	tgClient            *tg_client.Client
	s3                  *s3minio.MinioRepository
	repo                repository.Repository
	ggID                int
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
	repo repository.Repository,
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

func (tz *Tz) CheckTz(ctx context.Context, file []byte, filename string, userID uuid.UUID) (string, string, string, *[]OutInvalidError, *[]OutMissingError, string, error) {
	const op = "Tz.CheckTz"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("checking tz")

	timeStart := time.Now()

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

	log.Info("запрос промтов в promt-builder")

	//Генерируем запрос для нейронки
	neuralRequest, err := tz.promtBuilderClient.GeneratePromts(markdownResponse.Markdown, tz.ggID)
	if err != nil {
		return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", ErrLlmAnalyzeFile
	}

	if neuralRequest.Schema == nil {
		return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", ErrLlmAnalyzeFile
	}

	//if len(*neuralRequest.Items) > 1 {
	//	*neuralRequest.Items = (*neuralRequest.Items)[:1]
	//}

	groupReports := make([]tz_llm_client.GroupReport, 0, len(*neuralRequest.Items))

	allRubs := float64(0)
	allTokens := int64(0)

	// Создаем канал для результатов и waitgroup для синхронизации
	resultChan := make(chan llmRequestResult, len(*neuralRequest.Items))
	var wg sync.WaitGroup

	// Запускаем горутины для параллельной обработки запросов
	for _, v := range *neuralRequest.Items {
		wg.Add(1)
		go func(messages *[]struct {
			Role    *string `json:"role"`
			Content *string `json:"content"`
		}, schema map[string]interface{}) {
			defer wg.Done()

			// Гарантируем, что всегда отправляем результат в канал
			defer func() {
				if r := recover(); r != nil {
					log.Error("паника в goroutine: ", slog.Any("panic", r))
					resultChan <- llmRequestResult{err: fmt.Errorf("паника в goroutine: %v", r)}
				}
			}()

			llmResp, err := tz.llmClient.SendMessage(*messages, schema)
			if err != nil {
				log.Error("ошибка от llm request: ", sl.Err(err))
				resultChan <- llmRequestResult{err: err}
				return
			}

			if llmResp.Result == nil {
				log.Error("ошибка: в ответе от llm поле result пустое")
				resultChan <- llmRequestResult{err: fmt.Errorf("пустое поле result в ответе от llm")}
				return
			}

			result := llmRequestResult{
				groupReport: llmResp.Result,
			}

			if llmResp.Cost != nil {
				result.cost = llmResp.Cost.TotalRub
			}

			if llmResp.Usage != nil && llmResp.Usage.TotalTokens != nil {
				tokens := int64(*llmResp.Usage.TotalTokens)
				result.tokens = &tokens
			}

			resultChan <- result
		}(v.Messages, neuralRequest.Schema)
	}

	fmt.Println(" отладка 5")

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	fmt.Println(" отладка 6")

	// Собираем результаты
	expectedResults := len(*neuralRequest.Items)
	receivedResults := 0

	for result := range resultChan {
		receivedResults++
		fmt.Printf(" отладка: получен результат %d из %d\n", receivedResults, expectedResults)

		if result.err != nil {
			log.Error("ошибка в результате: ", sl.Err(result.err))
			continue
		}

		fmt.Println(" отладка 7")

		if result.groupReport != nil {
			groupReports = append(groupReports, *result.groupReport)
		}

		fmt.Println(" отладка 8")

		if result.cost != nil {
			allRubs += *result.cost
		}

		fmt.Println(" отладка 9")

		if result.tokens != nil {
			allTokens += *result.tokens
		}

		// Выходим из цикла, когда получили все ожидаемые результаты
		if receivedResults >= expectedResults {
			break
		}
	}

	fmt.Println(" отладка 10")

	inspectionTime := time.Since(timeStart)

	//llmAnalyzeResult, err := tz.llmClient.Analyze(markdownResponse.Markdown)

	//if err != nil {
	//	log.Error("Error: \n", err)
	//}
	//if llmAnalyzeResult == nil {
	//	//tz.tgClient.SendMessage("ИСПРАВИТЬ: от llm пришёл пустой ответ, но код ответа не ошибочный.")
	//
	//	log.Info("пустой ответ от llm")
	//	return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", ErrLlmAnalyzeFile
	//}
	//if llmAnalyzeResult.Reports == nil || len(llmAnalyzeResult.Reports) == 0 {
	//	//tz.tgClient.SendMessage("МБ ЧТО-ТО НЕ ТАК: от llm ответ без отчетов, но код ответа не ошибочный")
	//
	//	log.Info("0 отчетов в ответе от llm")
	//	return "", "", "", &([]OutInvalidError{}), &([]OutMissingError{}), "", ErrLlmAnalyzeFile
	//}

	outInvalidErrors, outMissingErrors, outHtml := HandleErrors(&groupReports, &markdownResponse.Mappings)

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
	err = tz.saveTechnicalSpecificationData(ctx, filename, userID, outHtml, *css, originalFileID, outInvalidErrors, outMissingErrors, allRubs, allTokens, inspectionTime, log)
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
	allRubs float64,
	allTokens int64,
	inspectionTime time.Duration,
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
		AllRubs:                  allRubs,
		AllTokens:                allTokens,
		InspectionTime:           inspectionTime,
	}

	version, err := tz.repo.CreateVersion(ctx, versionReq)
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}

	log.Info("version created", slog.String("version_id", version.ID.String()))

	// Сохраняем InvalidErrors
	if invalidErrors != nil && len(*invalidErrors) > 0 {
		invalidErrorData := make([]repository.InvalidErrorData, 0, len(*invalidErrors))
		for i, err := range *invalidErrors {
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
				OrderNumber:  i, // Порядковый номер (индекс в массиве)
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

func (tz *Tz) GetAllVersions(ctx context.Context) ([]*repository.VersionWithErrorCounts, error) {
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

func (tz *Tz) GetVersionStatistics(ctx context.Context) (*repository.VersionStatistics, error) {
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
	errorIDInt, err := strconv.Atoi(errorID)
	if err != nil {
		log.Error("invalid error ID format", slog.String("error", err.Error()))
		return fmt.Errorf("invalid error ID format: %w", err)
	}

	errorUUID, err := tz.repo.GetUUIDByErrorID(ctx, errorIDInt)
	if err != nil {
		log.Error("failed to get UUID by error ID", slog.String("error", err.Error()))
		return fmt.Errorf("failed to get UUID by error ID: %w", err)
	}

	// Определяем тип ошибки
	var repoErrorType repository.ErrorType
	switch errorType {
	case "invalid":
		repoErrorType = repository.ErrorTypeInvalid
	case "missing":
		repoErrorType = repository.ErrorTypeMissing
	default:
		log.Error("invalid error type", slog.String("errorType", errorType))
		return fmt.Errorf("invalid error type: %s", errorType)
	}

	now := time.Now()

	// Создаем запрос для создания обратной связи об ошибке
	feedbackReq := &repository.CreateErrorFeedbackRequest{
		ID:           uuid.New(),
		VersionID:    versionID,
		ErrorID:      errorUUID,
		ErrorType:    repoErrorType,
		UserID:       userID,
		FeedbackType: int(feedbackType),
		Comment:      &comment,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Если комментарий пустой, устанавливаем nil
	if comment == "" {
		feedbackReq.Comment = nil
	}

	// Сохраняем обратную связь в БД
	_, err = tz.repo.CreateErrorFeedback(ctx, feedbackReq)
	if err != nil {
		log.Error("failed to create error feedback", slog.String("error", err.Error()))
		return fmt.Errorf("failed to create error feedback: %w", err)
	}

	log.Info("feedback error created successfully")
	return nil
}
