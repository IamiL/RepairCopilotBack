package tzservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"html"
	"log/slog"
	"regexp"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
	"strconv"
	"strings"
)

type Tz struct {
	log                 *slog.Logger
	wordConverterClient *word_parser_client.Client
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
	Id    int
	Title string
	Text  string
	Type  string
}

func New(
	log *slog.Logger,
	wordConverterClient *word_parser_client.Client,
	llmClient *tz_llm_client.Client,
	tgClient *tg_client.Client,
	s3 *s3minio.MinioRepository,
) *Tz {
	return &Tz{
		log:                 log,
		wordConverterClient: wordConverterClient,
		llmClient:           llmClient,
		tgClient:            tgClient,
		s3:                  s3,
	}
}

func (tz *Tz) CheckTz(ctx context.Context, file []byte, filename string, requestID uuid.UUID) (string, string, string, []TzError, []TzError, string, error) {
	const op = "Tz.CheckTz"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("requestID", requestID.String()),
	)

	log.Info("checking tz")

	htmlText, css, err := tz.wordConverterClient.Convert(file, filename)
	if err != nil {
		tz.log.Info("Ошибка обработки файла в wordConverterClient: %v\n" + err.Error())

		tz.tgClient.SendMessage("Ошибка обработки файла в wordConverterClient: %v\n" + err.Error())

		return "", "", "", []TzError{}, []TzError{}, "", ErrConvertWordFile
	}

	log.Info("конвертация word файла в htmlText успешна")

	log.Info("отправка HTML файла в телеграм")

	htmlFileName := strings.TrimSuffix(filename, ".docx") + ".html"
	htmlFileData := []byte(*htmlText)
	err = tz.tgClient.SendFile(htmlFileData, htmlFileName)
	if err != nil {
		log.Error("ошибка отправки HTML файла в телеграм: ", sl.Err(err))
		tz.tgClient.SendMessage(fmt.Sprintf("Ошибка отправки HTML файла в телеграм: %v", err))
	} else {
		log.Info("HTML файл успешно отправлен в телеграм")
	}

	result, err := tz.llmClient.Analyze(*htmlText)
	if err != nil {
		log.Error("Error: \n", err)
	}
	if result == nil {
		tz.tgClient.SendMessage("ИСПРАВИТЬ: от llm пришёл пустой ответ, но код ответа не ошибочный.")

		log.Info("пустой ответ от llm")
		return "", "", "", []TzError{}, []TzError{}, "", ErrLlmAnalyzeFile
	}
	if result.Errors == nil {
		tz.tgClient.SendMessage("МБ ЧТО-ТО НЕ ТАК: от llm ответ без ошибок, но код ответа не ошибочный")

		log.Info("0 ошибок в ответе от llm")
		return "", "", "", []TzError{}, []TzError{}, "", ErrLlmAnalyzeFile
	}

	htmlTextResp := *htmlText

	errorsRespTemp := make([]TzError, 0, 100)

	errorId := 0

	for _, tzError := range result.Errors {
		for _, finding := range tzError.Findings {
			if len(finding.Quote) < 4 {
				continue
			}

			htmlTextResp = HighlightPhraseIgnoreCase(htmlTextResp, finding.Quote, errorId, finding.Paragraph)

			errorsRespTemp = append(errorsRespTemp, TzError{
				Id:    errorId,
				Title: tzError.Code + " " + tzError.Title,
				Text:  finding.Advice,
				Type:  "error",
			})

			errorId++
		}
	}

	//htmlTextResp = FixHTMLTags(htmlTextResp)

	//log.Info("ТЕКСТ НА ФРОНТ:")
	//log.Info(htmlTextResp)
	//log.Info("КОНЕЦ ТЕКСТА НА ФРОНТ")

	idsTemp := ExtractErrorIds(htmlTextResp)

	ids, err := StringsToInts(idsTemp)
	if err != nil {
		log.Error("ошибка преобразования массива ids_string в ids_int")
	}

	errorsResponse := SortByIdOrderFiltered(errorsRespTemp, ids)

	errorsMissingResponse := make([]TzError, 0, 100)

A:
	for _, tzError := range result.Errors {
		for _, finding := range tzError.Findings {

			if finding.Paragraph == "00000" {
				errorsMissingResponse = append(errorsMissingResponse, TzError{
					Id:    errorId,
					Title: tzError.Code + " " + tzError.Title,
					Text:  finding.Advice,
					Type:  "error",
				})
				continue A
			}

			errorId++
		}
	}

	log.Info("обращаемся к word-parser-service для преобразования в docx-файл с примечаниями")

	errorsMap := make(map[string]string, len(errorsResponse))

	for _, tzError := range errorsResponse {
		errorsMap[strconv.Itoa(tzError.Id)] = tzError.Title + " " + tzError.Text
	}

	file, err = tz.wordConverterClient.CreateDocumentFromHTML(htmlTextResp, errorsMap)
	if err != nil {
		log.Error("ошибка при обращении к  wordConverterClient: %v\n" + err.Error())
		return "", "", "", []TzError{}, []TzError{}, "", ErrGenerateDocxFile
	}

	log.Info("попытка сохранения docx-файла с примечаниями в s3")

	fileId, _ := uuid.NewUUID()

	err = tz.s3.SaveDocument(ctx, fileId.String(), file)
	if err != nil {
		log.Error("Error при сохранении docx-документа в s3: ", sl.Err(err))
	}

	log.Info("успешно сохранён файл в s3")

	log.Info("отправка файла в телеграм")
	err = tz.tgClient.SendFile(file, filename)
	if err != nil {
		log.Error("ошибка отправки файла в телеграм: ", sl.Err(err))
		tz.tgClient.SendMessage(fmt.Sprintf("Ошибка отправки файла в телеграм: %v", err))
	} else {
		log.Info("файл успешно отправлен в телеграм")
	}

	return htmlTextResp, *css, fileId.String(), errorsResponse, errorsMissingResponse, fileId.String(), nil

}

// HighlightPhraseIgnoreCase ищет фразу без учета регистра в указанном блоке
func HighlightPhraseIgnoreCase(text, phrase string, id int, blockNum string) string {
	if phrase == "" || blockNum == "" {
		return text
	}

	// Ищем блок с указанным номером
	blockPattern := fmt.Sprintf(`<[^>]*\b%s\b[^>]*>.*?</[^>]*>`, regexp.QuoteMeta(blockNum))
	blockRegex := regexp.MustCompile(blockPattern)

	// Находим блок
	blockMatch := blockRegex.FindString(text)
	if blockMatch == "" {
		return text // Блок не найден
	}

	blockStart := strings.Index(text, blockMatch)
	if blockStart == -1 {
		return text
	}

	lowerBlockContent := strings.ToLower(blockMatch)
	lowerPhrase := strings.ToLower(phrase)

	// Ищем фразу только в содержимом блока
	index := strings.Index(lowerBlockContent, lowerPhrase)
	if index == -1 {
		return text // Фраза не найдена в блоке
	}

	modifiedBlock := blockMatch

	// Заменяем все вхождения фразы в блоке
	for index != -1 {
		// Извлекаем оригинальную фразу с сохранением регистра
		originalPhrase := modifiedBlock[index : index+len(phrase)]
		escapedPhrase := html.EscapeString(originalPhrase)
		highlightedPhrase := fmt.Sprintf(`<span error-id="%d">%s</span>`, id, escapedPhrase)

		// Заменяем найденную фразу в блоке
		modifiedBlock = modifiedBlock[:index] + highlightedPhrase + modifiedBlock[index+len(phrase):]

		// Ищем следующее вхождение
		searchStart := index + len(highlightedPhrase)
		if searchStart >= len(modifiedBlock) {
			break
		}

		lowerModifiedBlock := strings.ToLower(modifiedBlock[searchStart:])
		nextIndex := strings.Index(lowerModifiedBlock, lowerPhrase)
		if nextIndex == -1 {
			break
		}
		index = searchStart + nextIndex
	}

	// Заменяем оригинальный блок на модифицированный в полном тексте
	result := strings.Replace(text, blockMatch, modifiedBlock, 1)

	return result
}

//func FixHTMLTags(input string) string {
//	// Регулярное выражение для открывающих тегов <p[числа]>
//	openTagRegex := regexp.MustCompile(`<p\d+>`)
//
//	// Регулярное выражение для закрывающих тегов </p[числа]>
//	closeTagRegex := regexp.MustCompile(`</p\d+>`)
//
//	// Заменяем открывающие теги
//	result := openTagRegex.ReplaceAllString(input, "<p>")
//
//	// Заменяем закрывающие теги
//	result = closeTagRegex.ReplaceAllString(result, "</p>")
//
//	return result
//}

// extractErrorIds извлекает все error-id из span тегов в тексте
func ExtractErrorIds(text string) []string {
	// Регулярное выражение для поиска <span error-id="...">
	// Поддерживает пробелы вокруг атрибутов и другие атрибуты
	re := regexp.MustCompile(`<span[^>]*\berror-id="([^"]+)"[^>]*>`)

	// Найти все совпадения с группами захвата
	matches := re.FindAllStringSubmatch(text, -1)

	// Извлечь значения id из групп захвата
	var ids []string
	for _, match := range matches {
		if len(match) > 1 {
			ids = append(ids, match[1])
		}
	}

	return ids
}

// StringsToInts преобразует массив строк в массив int
// Возвращает ошибку, если какая-то строка не является числом
func StringsToInts(strings []string) ([]int, error) {
	ints := make([]int, len(strings))

	for i, str := range strings {
		num, err := strconv.Atoi(str)
		if err != nil {
			return nil, fmt.Errorf("не удалось преобразовать '%s' в число: %v", str, err)
		}
		ints[i] = num
	}

	return ints, nil
}

// SortByIdOrderFiltered - альтернативная версия, которая возвращает только те элементы,
// ID которых есть во втором массиве, в точном порядке
func SortByIdOrderFiltered(responses []TzError, idOrder []int) []TzError {
	// Создаем map для быстрого поиска структур по ID
	idToResponse := make(map[int]TzError)
	for _, response := range responses {
		idToResponse[response.Id] = response
	}

	// Создаем результирующий массив в нужном порядке
	var result []TzError
	for _, id := range idOrder {
		if response, exists := idToResponse[id]; exists {
			result = append(result, response)
		}
	}

	return result
}
