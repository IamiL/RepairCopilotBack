package tzservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser2/paragraphs"

	//docxToDocx2007clientclient "repairCopilotBot/tz-bot/internal/pkg/docxToDocx2007client"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
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

	var tz_name string

	isDocFormat, err := IsDocFormat(filename)
	if err != nil {
		return uuid.Nil, "", time.Time{}, fmt.Errorf("ошибка сохранения файла в S3: %w", err)
	}

	if isDocFormat {
		tz_name = RemoveDocExtension(filename)
	} else {
		tz_name = RemoveDocxExtension(filename)
		//DocxToDocx2007ConverterClient, err := docxToDocx2007clientclient.New("localhost", 8000)
		//if err != nil || DocxToDocx2007ConverterClient == nil {
		//	if err != nil {
		//		log.Error("error initializing docx to docx 2007 converter client", sl.Err(err))
		//	}
		//	if DocxToDocx2007ConverterClient == nil {
		//		log.Error("error initializing docx to docx 2007 converter client")
		//	}
		//} else {
		//	newFile, err := DocxToDocx2007ConverterClient.Convert(ctx, file, filename)
		//	if err != nil {
		//		log.Error("error in convert docx to docx 2007 converter client", sl.Err(err))
		//	} else {
		//		file = newFile
		//	}
		//
		//}
	}

	// Инкрементируем счетчик проверок для пользователя (проверяем лимит)
	if tz.userServiceClient != nil {
		err = tz.userServiceClient.IncrementInspectionsForToday(ctx, userID.String())
		if err != nil {
			log.Error("failed to increment inspections for today", sl.Err(err))
			return uuid.Nil, "", time.Time{}, fmt.Errorf("inspection limit exceeded or user service error: %w", err)
		} else {
			log.Info("inspections counter incremented successfully")
		}
	}

	// Создаем техническую спецификацию
	newTzID := uuid.New()
	ts, err := tz.repo.CreateTechnicalSpecification(ctx, &modelrepo.CreateTechnicalSpecificationRequest{
		ID:        newTzID,
		Name:      tz_name,
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		tz.decrementInspectionsForUser(ctx, userID, log)
		return uuid.Nil, "", time.Time{}, fmt.Errorf("failed to create technical specification: %w", err)
	}
	log.Info("technical specification created", slog.String("ts_id", ts.ID.String()))

	originalFileName := tz_name + GetCurrentDateTimeString()
	// Сохраняем оригинальный файл в S3
	err = tz.s3.SaveDocument(ctx, originalFileName, file, "docs",
		func() string {
			if isDocFormat {
				return ".doc"
			} else {
				return ".docx"
			}
		}(),
	)
	if err != nil {
		log.Error("ошибка сохранения оригинального файла в S3: ", sl.Err(err))
		tz.decrementInspectionsForUser(ctx, userID, log)
		return uuid.Nil, "", time.Time{}, fmt.Errorf("ошибка сохранения файла в S3: %w", err)
	}
	log.Info("оригинальный файл успешно сохранён в S3", slog.String("file_id", originalFileName))

	// Создаем версию с минимальными данными и статусом "in_progress"
	newVersionID := uuid.New()
	versionReq := &modelrepo.CreateVersionRequest{
		ID:                       newVersionID,
		TechnicalSpecificationID: newTzID,
		VersionNumber:            1,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
		OriginalFileID:           originalFileName,
		OutHTML:                  "",
		CSS:                      "",
		CheckedFileID:            "",
		AllRubs:                  0,
		AllTokens:                0,
		InspectionTime:           0,
		OriginalFileSize:         int64(len(file)),
		NumberOfErrors:           0,
		Status:                   "in_progress",
		Progress:                 3,
	}
	err = tz.repo.CreateVersion(ctx, versionReq)
	if err != nil {
		tz.decrementInspectionsForUser(ctx, userID, log)
		return uuid.Nil, "", time.Time{}, fmt.Errorf("failed to create version: %w", err)
	}
	log.Info("version created with status 'in_progress'", slog.String("version_id", newVersionID.String()))

	// Запускаем асинхронную обработку
	go tz.ProcessTzAsync(file, filename, newVersionID, originalFileName, isDocFormat, tz_name, userID)

	log.Info("async processing started")
	return newVersionID, RemoveDocxExtension(filename), time.Now(), nil
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
	InvalidInstances    *[]OutInvalidError        `json:"invalid_instances,omitempty"`
	MissingInstances    *[]OutMissingError        `json:"missing_instances,omitempty"`
}

func (tz *Tz) ProcessTzAsync(file []byte, filename string, versionID uuid.UUID, _ string, isDocFormat bool, tzName string, userID uuid.UUID) {
	const op = "Tz.ProcessTzAsync"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("versionID", versionID.String()),
		slog.String("tzName", tzName),
		slog.String("userID", userID.String()),
	)

	log.Info("starting async processing")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Minute)
	defer cancel()

	//htmlText, css, err := tz.wordConverterClient.Convert(file, filename)
	//if err != nil {
	//	tz.log.Error("Ошибка обработки файла в wordConverterClient: " + err.Error())
	//	tz.updateVersionWithError(ctx, versionID, "error")
	//	return
	//}

	now := time.Now()

	if isDocFormat {
		newFile, err := tz.docToDocXConverterClient.Convert(file, filename)
		if err != nil {
			log.Error("ошибка при конвертации doc в docx: ", sl.Err(err))
		}

		file = newFile
	}

	oldVersion := false

	var paragraphs *string

	htmlWithPlaceholder := ""

	//html, err := tz.wordConverterClient2.Convert(file, filename)
	//if err != nil {
	//	log.Error("ошибка при обращении к wordParserClient2: ", sl.Err(err))
	//} else {
	//	respHtmlWithPlaceholdersStr, respParagraphsStr := paragraphsproc.ExtractParagraphs(html)
	//	paragraphs = &respParagraphsStr
	//	htmlWithPlaceholder = respHtmlWithPlaceholdersStr
	//resultExtractParagraphs := word_parser2.ExtractParagraphs(html)
	//paragraphs = &resultExtractParagraphs.Paragraphs
	//htmlWithPlaceholder = resultExtractParagraphs.HTMLWithPlaceholder
	//}
	//if paragraphs == nil || *paragraphs == "" {
	//	err = errors.New("failed to extract paragraphs")
	//}
	//if err != nil {
	//log.Error("ошибка при обращении к wordParserClient2: ", sl.Err(err))
	log.Info("пробуем старый word_parser")
	oldVersion = true
	paragraphsFromWordConverterClient, _, wordConverterClientErr := tz.wordConverterClient.Convert(file, RemoveDocExtension(filename)+".docx")
	if wordConverterClientErr != nil {
		tz.handleProcessingError(ctx, versionID, userID, "ошибка при обращении к wordParserClient: "+wordConverterClientErr.Error(), log)
		return
	}

	html := *paragraphsFromWordConverterClient

	paragraphs = paragraphsFromWordConverterClient
	//}

	log.Info("конвертация word файла в htmlText успешна")

	markdownResponse, err := tz.markdownClient.Convert(*paragraphs)
	if err != nil {
		tz.handleProcessingError(ctx, versionID, userID, "ошибка конвертации HTML в markdown: "+err.Error(), log)
		return
	}

	log.Info("конвертация HTML в markdown успешна")
	log.Info(fmt.Sprintf("получены дополнительные данные: message=%s, mappings_count=%d", markdownResponse.Message, len(markdownResponse.Mappings)))

	tz.mu.RLock()
	promts, schema, errorsDescrptions, err := tz.promtBuilderClient.GeneratePromts(markdownResponse.Markdown, tz.ggID)
	tz.mu.RUnlock()
	if err != nil {
		tz.handleProcessingError(ctx, versionID, userID, "ошибка генерации промтов: "+err.Error(), log)
		return
	}

	if schema == nil {
		tz.handleProcessingError(ctx, versionID, userID, "схема пустая", log)
		return
	}

	groupReports := make([]tz_llm_client.GroupReport, 0, len(*promts))
	allRubs := float64(0)
	allTokens := int64(0)

	// Создаем канал для результатов и waitgroup для синхронизации
	resultChan := make(chan llmRequestResult, len(*promts))
	var wg sync.WaitGroup

	progressNumberSteps := len(*promts) + 1
	progressOneStep := 100 / progressNumberSteps
	progressSteps := 0
	var progressStepsMu sync.RWMutex

	// Запускаем горутины для параллельной обработки запросов
	for _, v := range *promts {
		wg.Add(1)
		go func(messagesFromPromtBuilder *[]promt_builder.Message, schema json.RawMessage) {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					log.Error("паника в goroutine: ", slog.Any("panic", r))
					resultChan <- llmRequestResult{err: fmt.Errorf("паника в goroutine: %v", r)}
				}
			}()

			messages := make([]struct {
				Role    *string `json:"role"`
				Content *string `json:"content"`
			},
				0)

			for _, msg := range *messagesFromPromtBuilder {
				messages = append(messages, struct {
					Role    *string `json:"role"`
					Content *string `json:"content"`
				}{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}

			tz.mu.RLock()
			llmResp, err := tz.llmClient.SendMessage(messages, schema, 1, tz.useLlmCache)
			tz.mu.RUnlock()
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
				ResultRaw:   string(llmResp.ResultRaw),
				duration:    *llmResp.Duration,
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
		progressStepsMu.Lock()
		progressSteps += 1
		progressStepsMu.Unlock()

		progressStepsMu.RLock()
		go func() {
			UpdateVersionProgressErr := tz.repo.UpdateVersionProgress(ctx, versionID, progressSteps*progressOneStep)
			if UpdateVersionProgressErr != nil {
				log.Error("Error in UpdateVersionProgress: ", UpdateVersionProgressErr.Error())
			} else {
				fmt.Println("прогресс обновлён")
			}
		}()
		progressStepsMu.RUnlock()

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

	SortGroupReports(groupReports)

	rawGroupAnalizeResult, err := json.Marshal(groupReports)
	if err != nil {
		tz.handleProcessingError(ctx, versionID, userID, "ошибка rawGroupAnalizeResult: "+err.Error(), log)
		return
	}

	messagesFromPromtBuilder, step2schema, GenerateStep2PromtsErr := tz.promtBuilderClient.GenerateStep2Promts(string(rawGroupAnalizeResult), markdownResponse.Markdown)
	if GenerateStep2PromtsErr != nil {
		log.Error("GenerateStep2Promts error: " + GenerateStep2PromtsErr.Error())

		tz.handleProcessingError(ctx, versionID, userID, "ошибка messagesFromPromtBuilder: "+GenerateStep2PromtsErr.Error(), log)
		return
	}

	messages := make([]struct {
		Role    *string `json:"role"`
		Content *string `json:"content"`
	},
		0)

	for _, msg := range *messagesFromPromtBuilder {
		messages = append(messages, struct {
			Role    *string `json:"role"`
			Content *string `json:"content"`
		}{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	step2LlmResponse, step2LlmError := tz.llmClient.SendMessage(messages, step2schema, 2, tz.useLlmCache)
	if step2LlmError != nil {
		log.Error("step2Llm error: " + step2LlmError.Error())

		tz.handleProcessingError(ctx, versionID, userID, "ошибка step2Llm send message: "+step2LlmError.Error(), log)
		return
	}

	//llmReportRaw := string(step2LlmResponse.ResultRaw)
	//log.Info(llmReportRaw)

	//var llmReport LlmReport

	//step2LlmResponseMarshaled, step2LlmResponseMarshalErr := step2LlmResponse.ResultRaw.MarshalJSON()
	//if step2LlmResponseMarshalErr != nil {
	//	log.Error("step2LlmResponseMarshal error: " + step2LlmResponseMarshalErr.Error())
	//}
	//
	//UnmarshalLlmReportRawErr := json.Unmarshal(step2LlmResponseMarshaled, &llmReport)
	//if UnmarshalLlmReportRawErr != nil {
	//	log.Error("UnmarshalLlmReportRaw error: " + UnmarshalLlmReportRawErr.Error())
	//	return
	//}

	for i := range groupReports {
		for j := range *groupReports[i].Errors {
			(*groupReports[i].Errors)[j].ID = uuid.New()
		}
	}

	for i := range *step2LlmResponse.ResultStep2.Sections {
		for j := range *(*step2LlmResponse.ResultStep2.Sections)[i].FinalInstanceIds {
			for _, step1groupReport := range groupReports {
				for _, step1error := range *step1groupReport.Errors {
					for _, step1instance := range *step1error.Instances {
						if *step1instance.LlmId == (*(*step2LlmResponse.ResultStep2.Sections)[i].FinalInstanceIds)[j] {
							if (*step2LlmResponse.ResultStep2.Sections)[i].Instances == nil {
								insts := make([]tz_llm_client.LlmStep2Instance, 0)
								(*step2LlmResponse.ResultStep2.Sections)[i].Instances = &insts
							}

							errorID := step1error.ID.String()
							*(*step2LlmResponse.ResultStep2.Sections)[i].Instances = append(*(*step2LlmResponse.ResultStep2.Sections)[i].Instances, tz_llm_client.LlmStep2Instance{
								WhatIsIncorrect: step1instance.WhatIsIncorrect,
								Fix:             step1instance.Fix,
								ErrorID:         &errorID,
								Risks:           step1instance.Risks,
								Priority:        step1instance.Priority,
								LlmID:           step1instance.LlmId,
							})
						}
					}
				}
			}
		}
	}

	llmFinalReport, llmFinalReportMarshalErr := json.Marshal(*step2LlmResponse.ResultStep2)
	if llmFinalReportMarshalErr != nil {
		log.Error("llmFinalReport Marshal error: " + llmFinalReportMarshalErr.Error())
		tz.handleProcessingError(ctx, versionID, userID, "ошибка llmFinalReportMarshalErr: "+llmFinalReportMarshalErr.Error(), log)
		return
	}

	errorsInTz := ErrorsFormation(groupReports, errorsDescrptions)
	SortErrorsByCode(errorsInTz)
	for i := range errorsInTz {
		errorsInTz[i].OrderNumber = i
	}

	outInvalidErrors, outMissingErrors, htmlParagrapsWithWrappedErrors := HandleErrors(&groupReports, &markdownResponse.Mappings)

	var outHtml string

	if oldVersion {
		outHtml = htmlParagrapsWithWrappedErrors
	} else {
		//outHtml = word_parser2.InsertParagraphs(htmlWithPlaceholder, htmlParagrapsWithWrappedErrors)
		outHtml = paragraphsproc.InsertParagraphs(htmlWithPlaceholder, htmlParagrapsWithWrappedErrors)
	}

	for i := range *outInvalidErrors {
		(*outInvalidErrors)[i].OrderNumber = i
	}

	for i := range errorsInTz {
		invalidInstances := make([]OutInvalidError, 0)
		missingInstances := make([]OutMissingError, 0)
		for j := range *outInvalidErrors {
			if (*outInvalidErrors)[j].ErrorID == errorsInTz[i].ID {
				invalidInstances = append(invalidInstances, (*outInvalidErrors)[j])
			}
		}

		for j := range missingInstances {
			if missingInstances[j].ErrorID == errorsInTz[i].ID {
				missingInstances = append(missingInstances, missingInstances[j])
			}
		}

		errorsInTz[i].InvalidInstances = &invalidInstances
		errorsInTz[i].MissingInstances = &missingInstances
	}

	err = tz.repo.SaveInvalidInstances(ctx, outInvalidErrors)
	if err != nil {
		tz.handleProcessingError(ctx, versionID, userID, "ошибка сохранения invalid instances: "+err.Error(), log)
		return
	}

	err = tz.repo.SaveMissingInstances(ctx, outMissingErrors)
	if err != nil {
		tz.handleProcessingError(ctx, versionID, userID, "ошибка сохранения missing instances: "+err.Error(), log)
		return
	}

	invalidInstances2 := make([]OutInvalidError, 0)
	for i := range errorsInTz {
		invalidInstancesFromDb, err := tz.repo.GetInvalidInstancesByErrorID(ctx, errorsInTz[i].ID)
		if err != nil {
			log.Error("failed to get version errors: ", sl.Err(err))
		} else {
			for j := range *invalidInstancesFromDb {
				(*invalidInstancesFromDb)[j].HtmlIDStr = strconv.Itoa(int((*invalidInstancesFromDb)[j].HtmlID))
			}
			invalidInstances2 = append(invalidInstances2, *invalidInstancesFromDb...)
			errorsInTz[i].InvalidInstances = invalidInstancesFromDb
		}

		missingInstances, err := tz.repo.GetMissingInstancesByErrorID(ctx, errorsInTz[i].ID)
		if err != nil {
			log.Error("failed to get version missing instances: ", sl.Err(err))
		} else {
			errorsInTz[i].MissingInstances = missingInstances
		}
	}

	var reportFilename string

	reportCodument, err := tz.reportGeneratorClient.GenerateReport(ctx, step2LlmResponse.ResultStep2)
	if err != nil {
		log.Error("ошибка генерации docx-отчёта: ", sl.Err(err))
	} else {
		reportFilename = "отчёт_" + tzName + "_" + GetCurrentDateTimeString()
		err = tz.s3.SaveDocument(ctx, reportFilename, reportCodument, "reports", ".docx")
		if err != nil {
			log.Error("ошибка сохранения docx отчёта в s3: ", sl.Err(err))
		}
	}

	if errorsInTz != nil && len(errorsInTz) > 0 {
		errorData := make([]modelrepo.ErrorData, 0, len(errorsInTz))
		for _, err := range errorsInTz {
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

	inspectionTime := time.Since(now)

	// Обновляем версию с результатами обработки
	updateReq := &modelrepo.UpdateVersionRequest{
		ID:                              versionID,
		UpdatedAt:                       time.Now(),
		OutHTML:                         outHtml,
		CSS:                             "",
		CheckedFileID:                   reportFilename,
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
		LlmReport:                       string(llmFinalReport),
	}
	err = tz.repo.UpdateVersion(ctx, updateReq)
	if err != nil {
		tz.handleProcessingError(ctx, versionID, userID, "ошибка обновления версии: "+err.Error(), log)
		return
	}

	log.Info("async processing completed successfully")
}

func SortGroupReports(reports []tz_llm_client.GroupReport) {
	sort.Slice(reports, func(i, j int) bool {
		gi, gj := reports[i].GroupID, reports[j].GroupID
		switch {
		case gi == nil && gj == nil:
			return false // равны — порядок не меняем
		case gi == nil:
			return false // nil после не-nil
		case gj == nil:
			return true // не-nil перед nil
		default:
			return *gi < *gj // обычное сравнение
		}
	})
}

//type LlmReport struct {
//	Sections *[]Section `json:"sections"`
//	Notes    *string    `json:"notes"`
//}
//
//type Instance struct {
//	WhatSsIncorrect *string `json:"what_is_incorrect"`
//	Fix             *string `json:"how_to_fix"`
//	ErrorID         *string `json:"error_id"`
//}
//
//type Section struct {
//	ExistsInDoc        *bool       `json:"exists_in_doc"`
//	InitialInstanceIds *[]string   `json:"initial_instance_ids"`
//	FinalInstanceIds   *[]string   `json:"final_instance_ids"`
//	PartName           *string     `json:"part"`
//	Instances          *[]Instance `json:"instances"`
//}

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

// handleProcessingError обрабатывает ошибку в асинхронной обработке:
// 1. Обновляет статус версии на "error"
// 2. Декрементирует счетчик проверок пользователя
func (tz *Tz) handleProcessingError(ctx context.Context, versionID uuid.UUID, userID uuid.UUID, errorMsg string, log *slog.Logger) {
	log.Error(errorMsg)

	// Обновляем статус версии на "error"
	tz.updateVersionWithError(ctx, versionID, "error")

	// Декрементируем счетчик проверок пользователя
	if tz.userServiceClient != nil {
		err := tz.userServiceClient.DecrementInspectionsForToday(ctx, userID.String())
		if err != nil {
			log.Error("failed to decrement inspections for today", sl.Err(err))
		} else {
			log.Info("inspections counter decremented due to processing error")
		}
	}
}

// decrementInspectionsForUser декрементирует счетчик проверок пользователя
func (tz *Tz) decrementInspectionsForUser(ctx context.Context, userID uuid.UUID, log *slog.Logger) {
	if tz.userServiceClient != nil {
		err := tz.userServiceClient.DecrementInspectionsForToday(ctx, userID.String())
		if err != nil {
			log.Error("failed to decrement inspections for today", sl.Err(err))
		} else {
			log.Info("inspections counter decremented due to early error")
		}
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

// RemoveDocxExtension удаляет расширение ".doc" из конца строки (регистронезависимо)
func RemoveDocExtension(filename string) string {
	// Проверяем, заканчивается ли строка на ".docx" (регистронезависимо)
	if strings.HasSuffix(strings.ToLower(filename), ".doc") {
		return filename[:len(filename)-4]
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

func IsDocFormat(filename string) (bool, error) {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".doc") && !strings.HasSuffix(lower, ".docx") {
		return true, nil
	}
	if strings.HasSuffix(lower, ".docx") {
		return false, nil
	}
	return false, fmt.Errorf("file must have .doc or .docx extension")
}

func GetCurrentDateTimeString() string {
	now := time.Now()
	return fmt.Sprintf("%d.%d.%d.%02d.%02d.%02d.%02d",
		now.Day(),
		int(now.Month()),
		now.Year(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Nanosecond()/1000000)
}
