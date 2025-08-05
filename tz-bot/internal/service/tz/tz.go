package tzservice

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/agnivade/levenshtein"
	"github.com/google/uuid"
	"golang.org/x/net/html"
	"log/slog"
	"regexp"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
	"strings"
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
	NormText      string // нормализованный текст без тегов
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

		//tz.tgClient.SendMessage("Ошибка обработки файла в wordConverterClient: %v\n" + err.Error())

		return "", "", "", []TzError{}, []TzError{}, "", ErrConvertWordFile
	}

	log.Info("конвертация word файла в htmlText успешна")

	log.Info("отправляем HTML в markdown-service для конвертации")

	markdownResponse, err := tz.markdownClient.Convert(*htmlText)
	if err != nil {
		log.Error("ошибка конвертации HTML в markdown: ", sl.Err(err))
		//tz.tgClient.SendMessage(fmt.Sprintf("Ошибка конвертации HTML в markdown: %v", err))
		return "", "", "", []TzError{}, []TzError{}, "", fmt.Errorf("ошибка конвертации HTML в markdown: %w", err)
	}

	log.Info("конвертация HTML в markdown успешна")
	log.Info(fmt.Sprintf("получены дополнительные данные: message=%s, mappings_count=%d", markdownResponse.Message, len(markdownResponse.Mappings)))

	mdLines, err := readLinesFromString(markdownResponse.Markdown)
	if err != nil {
		log.Error("error in  readLinesFromString: ", err.Error())
		return "", "", "", nil, nil, "", err
	}
	//log.Info("отправка HTML файла в телеграм")
	//
	//htmlFileName := strings.TrimSuffix(filename, ".docx") + ".html"
	//htmlFileData := []byte(*htmlText)
	//err = tz.tgClient.SendFile(htmlFileData, htmlFileName)
	//if err != nil {
	//	log.Error("ошибка отправки HTML файла в телеграм: ", sl.Err(err))
	//	//tz.tgClient.SendMessage(fmt.Sprintf("Ошибка отправки HTML файла в телеграм: %v", err))
	//} else {
	//	log.Info("HTML файл успешно отправлен в телеграм")
	//}

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
		return "", "", "", []TzError{}, []TzError{}, "", ErrLlmAnalyzeFile
	}
	if llmAnalyzeResult.Reports == nil || len(llmAnalyzeResult.Reports) == 0 {
		//tz.tgClient.SendMessage("МБ ЧТО-ТО НЕ ТАК: от llm ответ без отчетов, но код ответа не ошибочный")

		log.Info("0 отчетов в ответе от llm")
		return "", "", "", []TzError{}, []TzError{}, "", ErrLlmAnalyzeFile
	}

	var instances []ErrorInstance
	for _, grp := range llmAnalyzeResult.Reports {
		for _, er := range grp.Errors {
			for _, inst := range er.Instances {
				instances = append(instances, ErrorInstance{
					GroupID:      grp.GroupID,
					Code:         er.Code,
					ErrType:      inst.ErrType,
					Snippet:      inst.Snippet,
					LineStart:    inst.LineStart,
					LineEnd:      inst.LineEnd,
					SuggestedFix: inst.SuggestedFix,
					Rationale:    inst.Rationale,
				})
			}
		}
	}

	// Для каждого блока заполняем NormText
	for i := range markdownResponse.Mappings {
		markdownResponse.Mappings[i].NormText = extractAndNormalize(markdownResponse.Mappings[i].HtmlContent)
	}

	var outErrors []OutError

	for _, inst := range instances {
		// генерация уникального ID
		errID := uuid.New().String()
		outErrors = append(outErrors, OutError{
			ID:           errID,
			GroupID:      inst.GroupID,
			Code:         inst.Code,
			SuggestedFix: inst.SuggestedFix,
			Rationale:    inst.Rationale,
		})

		// логируем начало обработки инстанса
		log.Info("Processing error instance",
			slog.String("error_id", errID),
			slog.String("group_id", inst.GroupID),
			slog.String("code", inst.Code),
			slog.String("err_type", inst.ErrType),
			slog.String("snippet", inst.Snippet),
			slog.Int("line_start", getInt(inst.LineStart)),
			slog.Int("line_end", getInt(inst.LineEnd)),
		)

		// Определить диапазон строк
		start, end := 1, len(mdLines)
		if inst.LineStart != nil {
			start = *inst.LineStart
			if inst.LineEnd != nil {
				end = *inst.LineEnd
			} else {
				end = start
			}
		}
		log.Info("[Error %s/%s] Search lines %d-%d for snippet: %q", inst.GroupID, inst.Code, start, end, inst.Snippet)

		// Нормализовать сниппет
		normSnippet := normalize(inst.Snippet)

		// Искать по блокам
		wrapped := false
		for i := range markdownResponse.Mappings {
			blk := &markdownResponse.Mappings[i]

			log.Debug("Trying HTML block for snippet",
				slog.String("html_element_id", blk.HtmlElementId),
				slog.Int("blk_line_start", blk.MarkdownLineStart),
				slog.Int("blk_line_end", blk.MarkdownLineEnd),
			)

			if blk.MarkdownLineStart > end || blk.MarkdownLineEnd < start {
				log.Debug("Skipping block — вне диапазона строк",
					slog.String("html_element_id", blk.HtmlElementId),
				)
				continue
			}
			// Exact match
			idx := strings.Index(blk.NormText, normSnippet)
			method := "exact"
			if idx == -1 {
				// fuzzy fallback
				idx_temp, dist := fuzzyFind(blk.NormText, normSnippet)
				idx = idx_temp
				method = "fuzzy"
				log.Debug("Fuzzy match result",
					slog.String("html_element_id", blk.HtmlElementId),
					slog.Int("distance", dist),
				)
			}
			if idx >= 0 {
				log.Info("[+] Found in %s at pos %d (method=%s)", blk.HtmlElementId, idx, method)

				log.Info("Wrapped snippet in span",
					slog.String("error_id", errID),
					slog.String("html_element_id", blk.HtmlElementId),
					slog.Int("offset", idx),
					slog.String("match_method", method),
				)

				blk.HtmlContent = injectSpan(blk.HtmlContent, normSnippet, idx, errID)
				wrapped = true
				break
			}
		}
		if !wrapped {
			log.Info("[-] Not found snippet %q", inst.Snippet)
		}

		if !wrapped {
			log.Warn("Snippet not found in any HTML block",
				slog.String("error_id", errID),
				slog.String("snippet", inst.Snippet),
			)
		}
	}

	var sb strings.Builder
	for _, blk := range markdownResponse.Mappings {
		sb.WriteString(blk.HtmlContent)
		sb.WriteString("\n")
	}
	finalHTML := sb.String()

	invalidErrorsResponse := make([]TzError, len(outErrors))

	for i := range outErrors {
		if outErrors[i].SuggestedFix != nil {
			invalidErrorsResponse[i] = TzError{
				Id:    outErrors[i].ID,
				Title: outErrors[i].GroupID,
				Text:  *outErrors[i].SuggestedFix,
				Type:  outErrors[i].Code,
			}
		} else {
			invalidErrorsResponse[i] = TzError{
				Id:    outErrors[i].ID,
				Title: outErrors[i].GroupID,
				Text:  " ",
				Type:  outErrors[i].Code,
			}
		}
	}

	missingErrorsResponse := make([]TzError, 1)

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

	//return htmlTextResp, *css, fileId.String(), errorsResponse, errorsMissingResponse, fileId.String(), nil
	return finalHTML, *css, "123", invalidErrorsResponse, missingErrorsResponse, "123", nil
}

// readLinesFromString разбирает переданный Markdown-текст на строки.
// Возвращает срез строк и ошибку, если она возникла при сканировании (очень маловероятна для строки).
func readLinesFromString(md string) ([]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(md))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// extractAndNormalize извлекает текст из HTML и нормализует его
func extractAndNormalize(htmlStr string) string {
	var sb strings.Builder
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return normalize(sb.String())
}

// normalize убирает Markdown-разметку, пунктуацию и приводит к нижнему регистру
func normalize(s string) string {
	reMd := regexp.MustCompile(`\*\*|__|\[|\]|\([^)]*\)`)
	s = reMd.ReplaceAllString(s, "")
	s = strings.ToLower(s)
	s = strings.Trim(s, ` ,.-–—!?:;"'`)
	reSp := regexp.MustCompile(`\s+`)
	s = reSp.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
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

// fuzzyFind ищет ближайшее вхождение по Левенштейну и возвращает индекс и расстояние
func fuzzyFind(text, pat string) (int, int) {
	minDist := len(pat)
	minIdx := -1
	for i := 0; i+len(pat) <= len(text); i++ {
		segment := text[i : i+len(pat)]
		dist := levenshtein.ComputeDistance(segment, pat)
		if dist < minDist {
			minDist = dist
			minIdx = i
		}
		if dist == 0 {
			break
		}
	}
	return minIdx, minDist
}

// injectSpan оборачивает найденный HTML-фрагмент в span с data-error
func injectSpan(htmlStr, normSnippet string, normIdx int, errID string) string {
	spanStart := fmt.Sprintf(`<span data-error="%s">`, errID)
	spanEnd := `</span>`
	return strings.Replace(htmlStr, normSnippet, spanStart+normSnippet+spanEnd, 1)
}

func getInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
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
//func ExtractErrorIds(text string) []string {
//	// Регулярное выражение для поиска <span error-id="...">
//	// Поддерживает пробелы вокруг атрибутов и другие атрибуты
//	re := regexp.MustCompile(`<span[^>]*\berror-id="([^"]+)"[^>]*>`)
//
//	// Найти все совпадения с группами захвата
//	matches := re.FindAllStringSubmatch(text, -1)
//
//	// Извлечь значения id из групп захвата
//	var ids []string
//	for _, match := range matches {
//		if len(match) > 1 {
//			ids = append(ids, match[1])
//		}
//	}
//
//	return ids
//}

// StringsToInts преобразует массив строк в массив int
// Возвращает ошибку, если какая-то строка не является числом
//func StringsToInts(strings []string) ([]int, error) {
//	ints := make([]int, len(strings))
//
//	for i, str := range strings {
//		num, err := strconv.Atoi(str)
//		if err != nil {
//			return nil, fmt.Errorf("не удалось преобразовать '%s' в число: %v", str, err)
//		}
//		ints[i] = num
//	}
//
//	return ints, nil
//}

// ProcessInvalidErrors обрабатывает ошибки типа invalid из LLM ответа
// Возвращает обработанные ошибки и обновленный HTML текст с подсветкой
//func ProcessInvalidErrors(reports []tz_llm_client.Report, mappings []markdown_service_client.Mapping, htmlText string) ([]TzError, string, int) {
//	errorsRespTemp := make([]TzError, 0, 100)
//	htmlTextResp := htmlText
//	errorId := 0
//
//	for _, report := range reports {
//		for _, tzError := range report.Errors {
//			if tzError.Verdict != "error_present" {
//				continue
//			}
//
//			for _, instance := range tzError.Instances {
//				if instance.ErrType != "invalid" {
//					continue
//				}
//
//				if len(instance.Snippet) < 4 {
//					continue
//				}
//
//				// Ищем подходящие маппинги по номерам строк
//				var targetMappings []markdown_service_client.Mapping
//				if instance.LineStart != nil && instance.LineEnd != nil {
//					for _, mapping := range mappings {
//						if mapping.MarkdownLineStart <= *instance.LineStart &&
//							mapping.MarkdownLineEnd >= *instance.LineEnd {
//							targetMappings = append(targetMappings, mapping)
//						}
//					}
//				}
//
//				// Если не нашли по номерам строк, используем все маппинги
//				if len(targetMappings) == 0 {
//					targetMappings = mappings
//				}
//
//				// Ищем фразу из snippet в HTML контенте маппингов
//				found := false
//				blockNum := "00000"
//
//				for _, mapping := range targetMappings {
//					if searchPhraseInHTML(instance.Snippet, mapping.HtmlContent) {
//						found = true
//						blockNum = mapping.HtmlElementId
//						break
//					}
//				}
//
//				// Если нашли совпадение, подсвечиваем в HTML
//				if found {
//					htmlTextResp = HighlightPhraseIgnoreCase(htmlTextResp, instance.Snippet, errorId, blockNum)
//				}
//
//				// Добавляем ошибку в результат
//				errorsRespTemp = append(errorsRespTemp, TzError{
//					Id:    errorId,
//					Title: tzError.Code + " " + instance.ErrType,
//					Text:  instance.SuggestedFix + " " + instance.Rationale,
//					Type:  "error",
//				})
//
//				errorId++
//			}
//		}
//	}
//
//	return errorsRespTemp, htmlTextResp, errorId
//}

// searchPhraseInHTML ищет фразу из markdown в HTML контенте
// Учитывает различия в форматировании между markdown и HTML
//func searchPhraseInHTML(snippet, htmlContent string) bool {
//	if snippet == "" || htmlContent == "" {
//		return false
//	}
//
//	// Приводим к нижнему регистру для поиска без учета регистра
//	lowerSnippet := strings.ToLower(snippet)
//	lowerHTML := strings.ToLower(htmlContent)
//
//	// Удаляем HTML теги из контента для чистого текстового поиска
//	htmlWithoutTags := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(lowerHTML, "")
//
//	// Нормализуем пробелы и знаки препинания
//	normalizeText := func(text string) string {
//		// Заменяем множественные пробелы на один
//		text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
//		// Удаляем некоторые знаки препинания для более гибкого поиска
//		text = regexp.MustCompile(`[,.;:!?""''«»]`).ReplaceAllString(text, "")
//		return strings.TrimSpace(text)
//	}
//
//	normalizedSnippet := normalizeText(lowerSnippet)
//	normalizedHTML := normalizeText(htmlWithoutTags)
//
//	// Пробуем точное совпадение
//	if strings.Contains(normalizedHTML, normalizedSnippet) {
//		return true
//	}
//
//	// Пробуем поиск по словам (если фраза разбита HTML тегами)
//	snippetWords := strings.Fields(normalizedSnippet)
//	if len(snippetWords) > 1 {
//		// Проверяем, что все слова присутствуют в тексте
//		allWordsFound := true
//		for _, word := range snippetWords {
//			if len(word) > 2 && !strings.Contains(normalizedHTML, word) {
//				allWordsFound = false
//				break
//			}
//		}
//		if allWordsFound {
//			return true
//		}
//	}
//
//	return false
//}

// SortByIdOrderFiltered - альтернативная версия, которая возвращает только те элементы,
// ID которых есть во втором массиве, в точном порядке
//func SortByIdOrderFiltered(responses []TzError, idOrder []int) []TzError {
//	// Создаем map для быстрого поиска структур по ID
//	idToResponse := make(map[int]TzError)
//	for _, response := range responses {
//		idToResponse[response.Id] = response
//	}
//
//	// Создаем результирующий массив в нужном порядке
//	var result []TzError
//	for _, id := range idOrder {
//		if response, exists := idToResponse[id]; exists {
//			result = append(result, response)
//		}
//	}
//
//	return result
//}
