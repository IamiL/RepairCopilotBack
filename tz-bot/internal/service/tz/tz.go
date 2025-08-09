package tzservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
	"strings"

	"github.com/google/uuid"
)

type Tz struct {
	log                 *slog.Logger
	wordConverterClient *word_parser_client.Client
	markdownClient      *markdown_service_client.Client
	llmClient           *tz_llm_client.Client
	tgClient            *tg_client.Client
	s3                  *s3minio.MinioRepository
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
) *Tz {
	return &Tz{
		log:                 log,
		wordConverterClient: wordConverterClient,
		markdownClient:      markdownClient,
		llmClient:           llmClient,
		tgClient:            tgClient,
		s3:                  s3,
	}
}

// splitMessage разбивает длинное сообщение на части с умным делением по границам предложений
func (tz *Tz) splitMessage(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}

	var messages []string
	remaining := text

	for len(remaining) > maxLength {
		// Найти лучшую точку разрыва в пределах maxLength
		breakPoint := tz.findBestBreakPoint(remaining, maxLength)

		if breakPoint == -1 {
			// Если не нашли хорошую точку разрыва, режем по maxLength
			breakPoint = maxLength
		}

		messages = append(messages, remaining[:breakPoint])
		remaining = remaining[breakPoint:]

		// Удаляем ведущие пробелы в следующей части
		remaining = strings.TrimLeft(remaining, " \n\t")
	}

	// Добавляем оставшуюся часть
	if len(remaining) > 0 {
		messages = append(messages, remaining)
	}

	return messages
}

// findBestBreakPoint ищет лучшую точку для разрыва сообщения
func (tz *Tz) findBestBreakPoint(text string, maxLength int) int {
	if len(text) <= maxLength {
		return len(text)
	}

	// Приоритеты для точек разрыва (в порядке предпочтения):
	// 1. Конец предложения (. ! ?)
	// 2. Конец абзаца (\n\n)
	// 3. Перенос строки (\n)
	// 4. После запятой или точки с запятой
	// 5. Пробел

	searchText := text[:maxLength]

	// Ищем конец предложения
	sentenceEnders := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}
	bestPoint := -1

	for _, ender := range sentenceEnders {
		if idx := strings.LastIndex(searchText, ender); idx != -1 && idx > bestPoint {
			bestPoint = idx + len(ender)
		}
	}

	if bestPoint > maxLength/2 { // Используем только если точка разрыва не слишком рано
		return bestPoint
	}

	// Ищем двойной перенос строки (конец абзаца)
	if idx := strings.LastIndex(searchText, "\n\n"); idx != -1 && idx > maxLength/3 {
		return idx + 2
	}

	// Ищем перенос строки
	if idx := strings.LastIndex(searchText, "\n"); idx != -1 && idx > maxLength/3 {
		return idx + 1
	}

	// Ищем запятую или точку с запятой
	punctuation := []string{", ", "; "}
	for _, punct := range punctuation {
		if idx := strings.LastIndex(searchText, punct); idx != -1 && idx > maxLength/2 {
			if idx > bestPoint {
				bestPoint = idx + len(punct)
			}
		}
	}

	if bestPoint > maxLength/2 {
		return bestPoint
	}

	// Ищем последний пробел
	if idx := strings.LastIndex(searchText, " "); idx != -1 && idx > maxLength/3 {
		return idx + 1
	}

	// Если ничего не нашли, возвращаем -1
	return -1
}

func (tz *Tz) CheckTz(ctx context.Context, file []byte, filename string, requestID uuid.UUID) (string, string, string, *[]OutInvalidError, *[]OutMissingError, string, error) {
	const op = "Tz.CheckTz"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("requestID", requestID.String()),
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

	//htmlTextResp = FixHTMLTags(htmlTextResp)

	//log.Info("ТЕКСТ НА ФРОНТ:")
	//log.Info(htmlTextResp)
	//log.Info("КОНЕЦ ТЕКСТА НА ФРОНТ")

	//log.Info("обращаемся к word-parser-service для преобразования в docx-файл с примечаниями")

	//errorsMap := make(map[string]string, len(errorsResponse))
	//
	//for _, tzError := range errorsResponse {
	//	errorsMap[strconv.Itoa(tzError.Id)] = tzError.Title + " " + tzError.Text
	//}

	//file, err = tz.wordConverterClient.CreateDocumentFromHTML(htmlTextResp, errorsMap)
	//if err != nil {
	//	log.Error("ошибка при обращении к  wordConverterClient: %v\n" + err.Error())
	//	return "", "", "", []TzError{}, []TzError{}, "", ErrGenerateDocxFile
	//}

	//log.Info("попытка сохранения docx-файла с примечаниями в s3")

	//fileId, _ := uuid.NewUUID()

	//err = tz.s3.SaveDocument(ctx, fileId.String(), file)
	//if err != nil {
	//	log.Error("Error при сохранении docx-документа в s3: ", sl.Err(err))
	//}

	//log.Info("успешно сохранён файл в s3")

	//htmlFileData2 := []byte(htmlTextResp)
	//err = tz.tgClient.SendFile(htmlFileData2, "123")
	//if err != nil {
	//	log.Error("ошибка отправки HTML файла в телеграм: ", sl.Err(err))
	//	tz.tgClient.SendMessage(fmt.Sprintf("Ошибка отправки HTML файла в телеграм: %v", err))
	//} else {
	//	log.Info("HTML файл успешно отправлен в телеграм")
	//}
	//
	//log.Info("отправка файла в телеграм")
	//err = tz.tgClient.SendFile(file, filename)
	//if err != nil {
	//	log.Error("ошибка отправки файла в телеграм: ", sl.Err(err))
	//	tz.tgClient.SendMessage(fmt.Sprintf("Ошибка отправки файла в телеграм: %v", err))
	//} else {
	//	log.Info("файл успешно отправлен в телеграм")
	//}

	return outHtml, *css, "123", outInvalidErrors, outMissingErrors, "123", nil
}
