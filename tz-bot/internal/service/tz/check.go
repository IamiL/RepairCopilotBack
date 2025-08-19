package tzservice

import (
	"context"
	"fmt"
	"log/slog"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (tz *Tz) CheckTz(ctx context.Context, file []byte, filename string, userID uuid.UUID) (string, string, string, *[]Error, *[]OutInvalidError, *[]OutMissingError, string, error) {
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

		return "", "", "", nil, nil, nil, "", ErrConvertWordFile
	}

	log.Info("конвертация word файла в htmlText успешна")

	log.Info("отправляем HTML в markdown-service для конвертации")

	markdownResponse, err := tz.markdownClient.Convert(*htmlText)
	if err != nil {
		log.Error("ошибка конвертации HTML в markdown: ", sl.Err(err))
		//tz.tgClient.SendMessage(fmt.Sprintf("Ошибка конвертации HTML в markdown: %v", err))
		return "", "", "", nil, nil, nil, "", fmt.Errorf("ошибка конвертации HTML в markdown: %w", err)
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
		return "", "", "", nil, nil, nil, "", ErrLlmAnalyzeFile
	}

	if neuralRequest.Schema == nil {
		return "", "", "", nil, nil, nil, "", ErrLlmAnalyzeFile
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

	for i := range groupReports {
		for j := range *groupReports[i].Errors {
			(*groupReports[i].Errors)[j].ID = uuid.New()
		}
	}

	errors := ErrorsFormation(groupReports)

	outInvalidErrors, outMissingErrors, outHtml := HandleErrors(&groupReports, &markdownResponse.Mappings)

	LogOutInvalidErrors(log, outInvalidErrors, "После сортировки")

	// Сохраняем оригинальный файл в S3
	originalFileID := uuid.New().String()
	err = tz.s3.SaveDocument(ctx, originalFileID, file)
	if err != nil {
		log.Error("ошибка сохранения оригинального файла в S3: ", sl.Err(err))
		return "", "", "", nil, nil, nil, "", fmt.Errorf("ошибка сохранения файла в S3: %w", err)
	}

	log.Info("оригинальный файл успешно сохранён в S3", slog.String("file_id", originalFileID))

	// Сохраняем данные в БД
	err = tz.saveTechnicalSpecificationData(ctx, filename, userID, outHtml, *css, originalFileID, outInvalidErrors, outMissingErrors, &errors, allRubs, allTokens, inspectionTime, log)
	if err != nil {
		log.Error("ошибка сохранения данных в БД: ", sl.Err(err))
		// Не возвращаем ошибку, чтобы не блокировать ответ пользователю
	}

	return outHtml, *css, "123", &errors, outInvalidErrors, outMissingErrors, "123", nil
}

type Error struct {
	ID                  uuid.UUID                 `json:"id"`
	GroupID             string                    `json:"group_id"`
	ErrorCode           string                    `json:"error_code"`
	PreliminaryNotes    *string                   `json:"preliminary_notes"`
	OverallCritique     *string                   `json:"overall_critique"`
	Verdict             string                    `json:"verdict"`
	ProcessAnalysis     *string                   `json:"process_analysis"`
	ProcessCritique     *string                   `json:"process_critique"`
	ProcessVerification *string                   `json:"process_verification"`
	ProcessRetrieval    *[]string                 `json:"process_retrieval"`
	Instances           *[]tz_llm_client.Instance `json:"instances"`
}
