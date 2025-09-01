package tzservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	word_parser2 "repairCopilotBot/tz-bot/internal/pkg/word-parser2"
	modelrepo "repairCopilotBot/tz-bot/internal/repository/models"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (tz *Tz) CheckTz(ctx context.Context, file []byte, filename string, userID uuid.UUID) (uuid.UUID, string, time.Time, error) {
	const op = "Tz.CheckTz"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("checking tz - creating initial records")

	// Создаем техническую спецификацию
	newTzID := uuid.New()
	ts, err := tz.repo.CreateTechnicalSpecification(ctx, &modelrepo.CreateTechnicalSpecificationRequest{
		ID:        newTzID,
		Name:      RemoveDocxExtension(filename),
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return uuid.Nil, "", time.Time{}, fmt.Errorf("failed to create technical specification: %w", err)
	}
	log.Info("technical specification created", slog.String("ts_id", ts.ID.String()))

	// Сохраняем оригинальный файл в S3
	originalFileID := uuid.New().String()
	err = tz.s3.SaveDocument(ctx, originalFileID, file, "docs")
	if err != nil {
		log.Error("ошибка сохранения оригинального файла в S3: ", sl.Err(err))
		return uuid.Nil, "", time.Time{}, fmt.Errorf("ошибка сохранения файла в S3: %w", err)
	}
	log.Info("оригинальный файл успешно сохранён в S3", slog.String("file_id", originalFileID))

	// Создаем версию с минимальными данными и статусом "in_progress"
	newVersionID := uuid.New()
	versionReq := &modelrepo.CreateVersionRequest{
		ID:                       newVersionID,
		TechnicalSpecificationID: newTzID,
		VersionNumber:            1,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
		OriginalFileID:           originalFileID,
		OutHTML:                  "",
		CSS:                      "",
		CheckedFileID:            "",
		AllRubs:                  0,
		AllTokens:                0,
		InspectionTime:           0,
		OriginalFileSize:         int64(len(file)),
		NumberOfErrors:           0,
		Status:                   "in_progress",
	}
	err = tz.repo.CreateVersion(ctx, versionReq)
	if err != nil {
		return uuid.Nil, "", time.Time{}, fmt.Errorf("failed to create version: %w", err)
	}
	log.Info("version created with status 'in_progress'", slog.String("version_id", newVersionID.String()))

	// Запускаем асинхронную обработку
	go tz.ProcessTzAsync(file, filename, newVersionID, originalFileID)

	log.Info("async processing started")
	return newVersionID, RemoveDocxExtension(filename), time.Now(), nil
}

type ErrorsArray struct {
	Errors []Error `json:"errors"`
}

type Error struct {
	ID                  uuid.UUID
	GroupID             string
	ErrorCode           string                    `json:"error_code"`
	OrderNumber         int                       `json:"order_number"`
	Name                string                    `json:"name"`
	Description         string                    `json:"description"`
	Detector            string                    `json:"detector"`
	PreliminaryNotes    *string                   `json:"preliminary_notes"`
	OverallCritique     *string                   `json:"overall_critique"`
	Verdict             string                    `json:"verdict"`
	ProcessAnalysis     *string                   `json:"process_analysis"`
	ProcessCritique     *string                   `json:"process_critique"`
	ProcessVerification *string                   `json:"process_verification"`
	ProcessRetrieval    *[]string                 `json:"process_retrieval"`
	Instances           *[]tz_llm_client.Instance `json:"instances"`
	InvalidInstances    *[]OutInvalidError        `json:"invalid_instances"`
	MissingInstances    *[]OutMissingError        `json:"missing_instances"`
}

func (tz *Tz) ProcessTzAsync(file []byte, filename string, versionID uuid.UUID, _ string) {
	const op = "Tz.ProcessTzAsync"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("versionID", versionID.String()),
	)

	log.Info("starting async processing")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	timeStart := time.Now()

	//htmlText, css, err := tz.wordConverterClient.Convert(file, filename)
	//if err != nil {
	//	tz.log.Error("Ошибка обработки файла в wordConverterClient: " + err.Error())
	//	tz.updateVersionWithError(ctx, versionID, "error")
	//	return
	//}

	oldVersion := false

	var paragraphs *string

	htmlWithPlaceholder := ""

	html, _, err := tz.wordConverterClient2.Convert(file, filename)
	if err != nil {
		log.Error("ошибка при обращении к wordParserClient2: ", sl.Err(err))
	} else {
		resultExtractParagraphs := word_parser2.ExtractParagraphs(html)
		paragraphs = &resultExtractParagraphs.Paragraphs
		htmlWithPlaceholder = resultExtractParagraphs.HTMLWithPlaceholder
	}
	if paragraphs == nil || *paragraphs == "" {
		err = errors.New("failed to extract paragraphs")
	}
	if err != nil {
		log.Error("ошибка при обращении к wordParserClient2: ", sl.Err(err))
		log.Info("пробуем старый word_parser")
		oldVersion = true
		paragraphs, _, err = tz.wordConverterClient.Convert(file, filename)
		if err != nil {
			log.Error("ошибка при обращении к wordParserClient: ", sl.Err(err))
		}
	}

	log.Info("конвертация word файла в htmlText успешна")

	markdownResponse, err := tz.markdownClient.Convert(*paragraphs)
	if err != nil {
		log.Error("ошибка конвертации HTML в markdown: ", sl.Err(err))
		tz.updateVersionWithError(ctx, versionID, "error")
		return
	}

	log.Info("конвертация HTML в markdown успешна")
	log.Info(fmt.Sprintf("получены дополнительные данные: message=%s, mappings_count=%d", markdownResponse.Message, len(markdownResponse.Mappings)))

	promts, schema, errorsDescrptions, err := tz.promtBuilderClient.GeneratePromts(markdownResponse.Markdown, tz.ggID)
	if err != nil {
		log.Error("ошибка генерации промтов: ", sl.Err(err))
		tz.updateVersionWithError(ctx, versionID, "error")
		return
	}

	if schema == nil {
		log.Error("схема пустая")
		tz.updateVersionWithError(ctx, versionID, "error")
		return
	}

	groupReports := make([]tz_llm_client.GroupReport, 0, len(*promts))
	allRubs := float64(0)
	allTokens := int64(0)

	// Создаем канал для результатов и waitgroup для синхронизации
	resultChan := make(chan llmRequestResult, len(*promts))
	var wg sync.WaitGroup

	// Запускаем горутины для параллельной обработки запросов
	for _, v := range *promts {
		wg.Add(1)
		go func(messages *[]struct {
			Role    *string `json:"role"`
			Content *string `json:"content"`
		}, schema map[string]interface{}) {
			defer wg.Done()

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
		}(v.Messages, schema)
	}

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Собираем результаты
	expectedResults := len(*promts)
	receivedResults := 0

	for result := range resultChan {
		receivedResults++

		if result.err != nil {
			log.Error("ошибка в результате: ", sl.Err(result.err))
			continue
		}

		if result.groupReport != nil {
			groupReports = append(groupReports, *result.groupReport)
		}

		if result.cost != nil {
			allRubs += *result.cost
		}

		if result.tokens != nil {
			allTokens += *result.tokens
		}

		if receivedResults >= expectedResults {
			break
		}
	}

	inspectionTime := time.Since(timeStart)

	for i := range groupReports {
		for j := range *groupReports[i].Errors {
			(*groupReports[i].Errors)[j].ID = uuid.New()
		}
	}

	errors := ErrorsFormation(groupReports, errorsDescrptions)
	SortErrorsByCode(errors)
	for i := range errors {
		errors[i].OrderNumber = i
	}

	outInvalidErrors, outMissingErrors, htmlParagrapsWithWrappedErrors := HandleErrors(&groupReports, &markdownResponse.Mappings)

	var outHtml string

	if oldVersion {
		outHtml = htmlParagrapsWithWrappedErrors
	} else {
		outHtml = word_parser2.InsertParagraphs(htmlWithPlaceholder, htmlParagrapsWithWrappedErrors)
	}

	for i := range *outInvalidErrors {
		(*outInvalidErrors)[i].OrderNumber = i
	}

	for i := range errors {
		invalidInstances := make([]OutInvalidError, 0)
		missingInstances := make([]OutMissingError, 0)
		for j := range *outInvalidErrors {
			if (*outInvalidErrors)[j].ErrorID == errors[i].ID {
				invalidInstances = append(invalidInstances, (*outInvalidErrors)[j])
			}
		}

		for j := range missingInstances {
			if missingInstances[j].ErrorID == errors[i].ID {
				missingInstances = append(missingInstances, missingInstances[j])
			}
		}

		errors[i].InvalidInstances = &invalidInstances
		errors[i].MissingInstances = &missingInstances
	}

	docxReportID := ""
	client := NewClient("http://localhost:8050", 30*time.Second)
	response, err := client.GenerateDocument(ctx, ErrorsArray{Errors: errors})
	if err != nil {
		log.Error("ошибка генерации docx-отчёта: ", sl.Err(err))
	} else {
		docxReportID = uuid.New().String()
		err = tz.s3.SaveDocument(ctx, docxReportID, response.Data, "reports")
		if err != nil {
			log.Error("ошибка сохранения docx отчёта в s3: ", sl.Err(err))
		}

	}

	err = tz.repo.SaveInvalidInstances(ctx, outInvalidErrors)
	if err != nil {
		log.Error("ошибка сохранения invalid instances: ", sl.Err(err))
		tz.updateVersionWithError(ctx, versionID, "error")
		return
	}

	err = tz.repo.SaveMissingInstances(ctx, outMissingErrors)
	if err != nil {
		log.Error("ошибка сохранения missing instances: ", sl.Err(err))
		tz.updateVersionWithError(ctx, versionID, "error")
		return
	}

	if errors != nil && len(errors) > 0 {
		errorData := make([]modelrepo.ErrorData, 0, len(errors))
		for _, err := range errors {
			instancesJSON, jsonErr := json.Marshal(err.Instances)
			if jsonErr != nil {
				log.Error("ошибка сериализации instances: ", sl.Err(jsonErr))
				continue
			}

			errorData = append(errorData, modelrepo.ErrorData{
				ID:                  err.ID,
				GroupID:             &err.GroupID,
				ErrorCode:           &err.ErrorCode,
				OrderNumber:         &err.OrderNumber,
				Name:                &err.Name,
				Description:         &err.Description,
				Detector:            &err.Detector,
				PreliminaryNotes:    err.PreliminaryNotes,
				OverallCritique:     err.OverallCritique,
				Verdict:             &err.Verdict,
				ProcessAnalysis:     err.ProcessAnalysis,
				ProcessCritique:     err.ProcessCritique,
				ProcessVerification: err.ProcessVerification,
				ProcessRetrieval:    err.ProcessRetrieval,
				Instances:           instancesJSON,
			})
		}

		errorsReq := &modelrepo.CreateErrorsRequest{
			VersionID: versionID,
			Errors:    errorData,
		}

		err = tz.repo.CreateErrors(ctx, errorsReq)
		if err != nil {
			log.Error("ошибка сохранения errors: ", sl.Err(err))
		} else {
			log.Info("errors saved", slog.Int("count", len(errorData)))
		}
	}

	mappingsFromMarkdownServiceJSON := make([]byte, 1)
	mappingsFromMarkdownServiceJSON, mappingsFromMarkdownServiceJSONErr := json.Marshal(markdownResponse.Mappings)
	if mappingsFromMarkdownServiceJSONErr != nil {
		log.Error("ошибка сериализации mappingsFromMarkdownService: ", sl.Err(mappingsFromMarkdownServiceJSONErr))
	}

	promtsFromPromtBuilderJSON := make([]byte, 1)
	promtsFromPromtBuilderJSON, promtsFromPromtBuilderJSONErr := json.Marshal(promts)
	if promtsFromPromtBuilderJSONErr != nil {
		log.Error("ошибка сериализации promtsFromPromtBuilder: ", sl.Err(promtsFromPromtBuilderJSONErr))
	}

	groupReportsFromLlmJSON := make([]byte, 1)
	groupReportsFromLlmJSON, groupReportsFromLlmJSONErr := json.Marshal(groupReports)
	if groupReportsFromLlmJSONErr != nil {
		log.Error("ошибка сериализации groupReportsFromLlm: ", sl.Err(groupReportsFromLlmJSONErr))
	}

	// Обновляем версию с результатами обработки
	updateReq := &modelrepo.UpdateVersionRequest{
		ID:                              versionID,
		UpdatedAt:                       time.Now(),
		OutHTML:                         outHtml,
		CSS:                             "",
		CheckedFileID:                   docxReportID,
		AllRubs:                         allRubs,
		AllTokens:                       allTokens,
		InspectionTime:                  inspectionTime,
		NumberOfErrors:                  len(*outMissingErrors) + len(*outInvalidErrors),
		Status:                          "completed",
		HtmlFromWordParser:              html,
		HtmlWithPlacrholder:             htmlWithPlaceholder,
		HtmlParagraphs:                  *paragraphs,
		MarkdownFromMarkdownService:     markdownResponse.Markdown,
		HtmlWithIdsFromMarkdownService:  markdownResponse.HtmlWithIds,
		MappingsFromMarkdownService:     mappingsFromMarkdownServiceJSON,
		PromtsFromPromtBuilder:          promtsFromPromtBuilderJSON,
		GroupReportsFromLlm:             groupReportsFromLlmJSON,
		HtmlParagraphsWithWrappesErrors: htmlParagrapsWithWrappedErrors,
	}
	err = tz.repo.UpdateVersion(ctx, updateReq)
	if err != nil {
		log.Error("ошибка обновления версии: ", sl.Err(err))
		return
	}

	log.Info("async processing completed successfully")
}

func (tz *Tz) updateVersionWithError(ctx context.Context, versionID uuid.UUID, status string) {
	updateReq := &modelrepo.UpdateVersionRequest{
		ID:             versionID,
		UpdatedAt:      time.Now(),
		OutHTML:        "",
		CSS:            "",
		CheckedFileID:  "",
		AllRubs:        0,
		AllTokens:      0,
		InspectionTime: 0,
		NumberOfErrors: 0,
		Status:         status,
	}
	err := tz.repo.UpdateVersion(ctx, updateReq)
	if err != nil {
		tz.log.Error("failed to update version with error status", slog.String("versionID", versionID.String()), slog.Any("error", err))
	}
}

// RemoveDocxExtension удаляет расширение ".docx" из конца строки (регистронезависимо)
func RemoveDocxExtension(filename string) string {
	// Проверяем, заканчивается ли строка на ".docx" (регистронезависимо)
	if strings.HasSuffix(strings.ToLower(filename), ".docx") {
		return filename[:len(filename)-5]
	}
	return filename
}

// SortErrorsByCode сортирует массив ошибок по ErrorCode
// Ожидаемый формат: E + число + опциональная буква (E01, E12, E07, E01A, E03B)
// Коды с неправильным форматом помещаются в конец массива
func SortErrorsByCode(errors []Error) {
	sort.Slice(errors, func(i, j int) bool {
		codeI := errors[i].ErrorCode
		codeJ := errors[j].ErrorCode

		// Регулярное выражение для парсинга кода ошибки
		re := regexp.MustCompile(`^E(\d+)([A-Z]?)$`)

		matchI := re.FindStringSubmatch(codeI)
		matchJ := re.FindStringSubmatch(codeJ)

		// Если оба кода соответствуют формату
		if matchI != nil && matchJ != nil {
			// Сравниваем числовую часть
			numI, _ := strconv.Atoi(matchI[1])
			numJ, _ := strconv.Atoi(matchJ[1])

			if numI != numJ {
				return numI < numJ
			}

			// Если числа равны, сравниваем буквенную часть
			letterI := matchI[2]
			letterJ := matchJ[2]

			// Если у одного есть буква, а у другого нет, тот что без буквы идет первым
			if letterI == "" && letterJ != "" {
				return true
			}
			if letterI != "" && letterJ == "" {
				return false
			}

			// Если у обоих есть буквы или у обоих нет, сравниваем лексикографически
			return letterI < letterJ
		}

		// Если только первый код соответствует формату, он идет первым
		if matchI != nil && matchJ == nil {
			return true
		}

		// Если только второй код соответствует формату, он идет первым
		if matchI == nil && matchJ != nil {
			return false
		}

		// Если оба кода не соответствуют формату, сортируем лексикографически
		return codeI < codeJ
	})
}
