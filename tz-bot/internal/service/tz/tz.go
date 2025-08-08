package tzservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"repairCopilotBot/tz-bot/internal/pkg/tg"
	"repairCopilotBot/tz-bot/internal/pkg/word-parser"
	"repairCopilotBot/tz-bot/internal/repository/s3minio"
	"sort"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"golang.org/x/net/html"
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

	//mdLines, err := readLinesFromString(markdownResponse.Markdown)
	//if err != nil {
	//	log.Error("error in  readLinesFromString: ", err.Error())
	//	return "", "", "", nil, nil, "", err
	//}
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

	instances := make([]ErrorInstance, 0, len(llmAnalyzeResult.Reports))
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
	//
	//// Для каждого блока заполняем NormText
	//for i := range markdownResponse.Mappings {
	//	markdownResponse.Mappings[i].NormText = extractAndNormalize(markdownResponse.Mappings[i].HtmlContent)
	//}
	//
	//var outErrors []OutError
	//
	//for _, inst := range instances {
	//	if inst.ErrType != "invalid" || strings.TrimSpace(inst.Snippet) == "" {
	//		log.Info("Skip highlighting, empty or non-invalid snippet",
	//			slog.String("group_id", inst.GroupID),
	//			slog.String("code", inst.Code),
	//			slog.String("err_type", inst.ErrType),
	//		)
	//		continue
	//	}
	//
	//	// генерация уникального ID
	//	errID := uuid.New().String()
	//	outErrors = append(outErrors, OutError{
	//		ID:           errID,
	//		GroupID:      inst.GroupID,
	//		Code:         inst.Code,
	//		SuggestedFix: inst.SuggestedFix,
	//		Rationale:    inst.Rationale,
	//	})
	//
	//	// логируем начало обработки инстанса
	//	log.Info("Processing error instance",
	//		slog.String("error_id", errID),
	//		slog.String("group_id", inst.GroupID),
	//		slog.String("code", inst.Code),
	//		slog.String("err_type", inst.ErrType),
	//		slog.String("snippet", inst.Snippet),
	//		slog.Int("line_start", getInt(inst.LineStart)),
	//		slog.Int("line_end", getInt(inst.LineEnd)),
	//	)
	//
	//	// Определить диапазон строк
	//	start, end := 1, len(mdLines)
	//	if inst.LineStart != nil {
	//		start = *inst.LineStart
	//		if inst.LineEnd != nil {
	//			end = *inst.LineEnd
	//		} else {
	//			end = start
	//		}
	//	}
	//	log.Info("[Error %s/%s] Search lines %d-%d for snippet: %q", inst.GroupID, inst.Code, start, end, inst.Snippet)
	//
	//	// Нормализовать сниппет
	//	normSnippet := normalize(inst.Snippet)
	//	if normSnippet == "" {
	//		log.Warn("Normalized snippet is empty, skipping",
	//			slog.String("error_id", errID),
	//			slog.String("original_snippet", inst.Snippet),
	//		)
	//		continue
	//	}
	//
	//	// Искать по блокам
	//	wrapped := false
	//	for i := range markdownResponse.Mappings {
	//		blk := &markdownResponse.Mappings[i]
	//
	//		log.Debug("Trying HTML block for snippet",
	//			slog.String("html_element_id", blk.HtmlElementId),
	//			slog.Int("blk_line_start", blk.MarkdownLineStart),
	//			slog.Int("blk_line_end", blk.MarkdownLineEnd),
	//		)
	//
	//		if blk.MarkdownLineStart > end || blk.MarkdownLineEnd < start {
	//			log.Debug("Skipping block — вне диапазона строк",
	//				slog.String("html_element_id", blk.HtmlElementId),
	//			)
	//			continue
	//		}
	//		// Exact match
	//		// Exact match in normalized text
	//		idx := strings.Index(blk.NormText, normSnippet)
	//		//method := "exact"
	//		if idx == -1 {
	//			// fuzzy fallback
	//			idx_temp, dist := fuzzyFind(blk.NormText, normSnippet)
	//			idx = idx_temp
	//			//method = "fuzzy"
	//			log.Debug("Fuzzy match result",
	//				slog.String("html_element_id", blk.HtmlElementId),
	//				slog.Int("distance", dist),
	//			)
	//		}
	//		if idx >= 0 {
	//			// Теперь найдём в исходном HTML ту же порцию текста
	//			original := extractOriginalSnippet(blk.HtmlContent, blk.NormText, normSnippet, idx)
	//			if original != "" {
	//				newHTML, err := wrapSnippetInSpan(blk.HtmlContent, original, errID)
	//				if err != nil {
	//					log.Error("Failed to wrap snippet", sl.Err(err))
	//				} else {
	//					blk.HtmlContent = newHTML
	//				}
	//				log.Info("Wrapped snippet in span", slog.String("error_id", errID),
	//					slog.String("html_element_id", blk.HtmlElementId),
	//					slog.String("original", original),
	//				)
	//				wrapped = true
	//				break
	//			}
	//		}
	//	}
	//	if !wrapped {
	//		log.Info("[-] Not found snippet %q", inst.Snippet)
	//	}
	//
	//	if !wrapped {
	//		log.Warn("Snippet not found in any HTML block",
	//			slog.String("error_id", errID),
	//			slog.String("snippet", inst.Snippet),
	//		)
	//	}
	//}
	//
	//var sb strings.Builder
	//for _, blk := range markdownResponse.Mappings {
	//	sb.WriteString(blk.HtmlContent)
	//	sb.WriteString("\n")
	//}
	//finalHTML := sb.String()
	//
	//invalidErrorsResponse := make([]TzError, len(outErrors))
	//
	//for i := range outErrors {
	//	if outErrors[i].SuggestedFix != nil {
	//		invalidErrorsResponse[i] = TzError{
	//			Id:    outErrors[i].ID,
	//			Title: outErrors[i].GroupID,
	//			Text:  *outErrors[i].SuggestedFix,
	//			Type:  outErrors[i].Code,
	//		}
	//	} else {
	//		invalidErrorsResponse[i] = TzError{
	//			Id:    outErrors[i].ID,
	//			Title: outErrors[i].GroupID,
	//			Text:  " ",
	//			Type:  outErrors[i].Code,
	//		}
	//	}
	//}
	//
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

	outHtml, invalidErrors, err := tz.integrateErrors(ctx, markdownResponse.HtmlWithIds, markdownResponse.Mappings, instances, log)
	if err != nil {
		log.Error("Ошибка алгоритма совмещения ошибок с html: ", sl.Err(err))
	}

	outInvalidErrors := make([]TzError, len(invalidErrors))

	for i := range invalidErrors {
		if invalidErrors[i].SuggestedFix != nil {
			outInvalidErrors[i] = TzError{
				Id:    invalidErrors[i].ID,
				Title: invalidErrors[i].GroupID,
				Text:  *invalidErrors[i].SuggestedFix,
				Type:  invalidErrors[i].Code,
			}
		} else {
			outInvalidErrors[i] = TzError{
				Id:    invalidErrors[i].ID,
				Title: invalidErrors[i].GroupID,
				Text:  " ",
				Type:  invalidErrors[i].Code,
			}
		}
	}

	return outHtml, *css, "123", outInvalidErrors, missingErrorsResponse, "123", nil
}

type blockWork struct {
	Map   markdown_service_client.Mapping
	Plain plainIndex
	HTML  string
	Repls [][3]int // список замен: [startByte, endByte, localIndex]
}

func (tz *Tz) integrateErrors(
	ctx context.Context,
	htmlWithIds string,
	mappings []markdown_service_client.Mapping,
	errors []ErrorInstance,
	log *slog.Logger,
) (string, []OutError, error) {

	// 1) подготовим блоки
	blocks := make(map[string]*blockWork, len(mappings))
	for i := range mappings {
		m := mappings[i]
		pi, err := buildPlainAndIndexFromHTML(m.HtmlContent)
		if err != nil {
			log.Error("plain/index build failed", "id", m.ElementID, "err", err)
			continue
		}
		blocks[m.ElementID] = &blockWork{
			Map:   m,
			Plain: pi,
			HTML:  m.HtmlContent,
		}
	}

	var out []OutError
	errCounter := 0

	// 2) обработка ошибок
	for _, inst := range errors {
		if inst.ErrType == "missing" {
			// нет вставок в HTML, просто регистрируем
			id := fmt.Sprintf("missing-%03d", errCounter)
			errCounter++
			out = append(out, OutError{
				ID: id, GroupID: inst.GroupID, Code: inst.Code,
				SuggestedFix: inst.SuggestedFix, Rationale: inst.Rationale,
			})
			log.Info("missing error recorded", "id", id, "code", inst.Code, "group", inst.GroupID)
			continue
		}

		if inst.ErrType != "invalid" {
			continue
		}

		// 2.1 кандидаты по линиям
		var candidates []*blockWork
		for _, bl := range blocks {
			if inst.LineStart != nil && inst.LineEnd != nil {
				if !(bl.Map.MarkdownEnd < *inst.LineStart || bl.Map.MarkdownStart > *inst.LineEnd) {
					candidates = append(candidates, bl)
				}
			} else {
				candidates = append(candidates, bl)
			}
		}
		if len(candidates) == 0 {
			candidates = mapValues(blocks)
		}

		// 2.2 ищем лучшее совпадение
		best := struct {
			b *blockWork
			m match
		}{}
		for _, b := range candidates {
			m := findBestMatch(inst.Snippet, b.Plain.Plain)
			if m.Found && (best.b == nil || m.Score > best.m.Score) {
				best = struct {
					b *blockWork
					m match
				}{b, m}
			}
		}

		id := fmt.Sprintf("e-%03d", errCounter)
		errCounter++

		if best.b == nil {
			log.Warn("invalid not matched", "code", inst.Code, "group", inst.GroupID, "snippet", inst.Snippet)
			// всё равно добавим OutError, чтобы фронт видел ошибку
			out = append(out, OutError{
				ID: id, GroupID: inst.GroupID, Code: inst.Code,
				SuggestedFix: inst.SuggestedFix, Rationale: inst.Rationale,
			})
			continue
		}

		// 2.3 проекция на HTML
		// защитимся от выхода за карту, берём ближайшие индексы
		startPlain := clamp(best.m.Start, 0, len(best.b.Plain.IndexMap)-1)
		endPlain := clamp(best.m.End-1, 0, len(best.b.Plain.IndexMap)-1)
		startByte := best.b.Plain.IndexMap[startPlain]
		endByte := best.b.Plain.IndexMap[endPlain] + 1 // символ включительно

		// 2.4 фиксация замены (отложенное применение)
		best.b.Repls = append(best.b.Repls, [3]int{startByte, endByte, len(best.b.Repls)})

		out = append(out, OutError{
			ID: id, GroupID: inst.GroupID, Code: inst.Code,
			SuggestedFix: inst.SuggestedFix, Rationale: inst.Rationale,
		})
		log.Info("invalid matched",
			"id", id, "element", best.b.Map.ElementID,
			"mscore", best.m.Score, "plainStart", startPlain, "plainEnd", endPlain,
			"byteStart", startByte, "byteEnd", endByte,
		)
	}

	// 3) применяем вставки (в каждом блоке — с конца)
	for _, b := range blocks {
		if len(b.Repls) == 0 {
			continue
		}
		sort.Slice(b.Repls, func(i, j int) bool { return b.Repls[i][0] > b.Repls[j][0] })

		html := b.HTML
		for _, r := range b.Repls {
			var err error
			html, err = wrapHTMLSpan(html, r[0], r[1], fmt.Sprintf("blk-%s-%d", b.Map.ElementID, r[2]))
			if err != nil {
				log.Error("wrap failed", "id", b.Map.ElementID, "err", err)
			}
		}
		b.HTML = html
	}

	// 4) пересобираем итоговый htmlWithIds (заменами по data-mapping-id)
	// Самый надёжный путь — через goquery: найти элемент с data-mapping-id и заменить html на b.HTML.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlWithIds))
	if err != nil {
		return htmlWithIds, out, err
	}

	for _, b := range blocks {
		sel := fmt.Sprintf(`[data-mapping-id="%s"]`, b.Map.ElementID)
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			// заменяем весь outer? у нас b.HTML включает сам элемент;
			// проще заменить через Parent? Или заменить s.SetHtml(inner) если b.HTML — innerHTML
			// В твоём json b.HtmlContent — полный outer HTML узла (p/li/ul). Делаем ReplaceWithHtml.
			s.ReplaceWithHtml(b.HTML)
		})
	}

	finalHTML, err := doc.Html()
	if err != nil {
		return htmlWithIds, out, err
	}

	return finalHTML, out, nil
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func mapValues[K comparable, V any](m map[K]V) []V {
	a := make([]V, 0, len(m))
	for _, v := range m {
		a = append(a, v)
	}
	return a
}

// помощники: нормализация/токенизация
func normalizeText(s string) string {
	// 1) HTML entities -> символы
	s = html.UnescapeString(s)
	// 2) заменить NBSP и прочие хитрые пробелы на обычный
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\u2009", " ")
	s = strings.ReplaceAll(s, "\u202F", " ")
	// 3) умные кавычки/тире -> обычные
	repl := map[string]string{
		"“": "\"", "”": "\"", "«": "\"", "»": "\"", "‘": "'", "’": "'",
		"—": "-", "–": "-",
	}
	for k, v := range repl {
		s = strings.ReplaceAll(s, k, v)
	}
	// 4) схлопываем пробелы
	s = strings.TrimSpace(s)
	reSpaces := regexp.MustCompile(`\s+`)
	s = reSpaces.ReplaceAllString(s, " ")
	return s
}

func stripMarkdown(s string) string {
	// очень лёгкая чистка markdown (для сниппетов):
	// **bold**, *i*, __u__, `code`, [text](url), начальные "- " / "* " списков
	s = regexp.MustCompile("`([^`]*)`").ReplaceAllString(s, "$1")
	s = strings.NewReplacer("**", "", "__", "", "*", "", "_", "").Replace(s)
	s = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`(?m)^\s*[-*]\s+`).ReplaceAllString(s, "")
	return s
}

type plainIndex struct {
	// mapping from plain-text index -> byte offset in original HTML
	// we будем хранить только границы, нам хватит массива длиной len(plain)+1,
	// где indexMap[i] = байтовая позиция в HTML до символа plain[i]
	IndexMap []int
	Plain    string
}

// вытащить текст из HTML и построить индексную карту
func buildPlainAndIndexFromHTML(htmlBlock string) (plainIndex, error) {
	// Парсим HTML как фрагмент
	node, err := html.Parse(strings.NewReader(htmlBlock))
	if err != nil {
		return plainIndex{}, err
	}

	var buf strings.Builder
	var indexMap []int
	var walk func(*html.Node, int) int
	walk = func(n *html.Node, htmlPos int) int {
		// htmlPos — текущее смещение по исходной HTML-строке (байты).
		// Для простоты считаем, что htmlBlock — исходная строка;
		// будем инкрементировать htmlPos, когда пишем в buf.
		// Реализуем через повторный рендер ноды до текста? Проще: пробежимся текст-ноды.
		if n.Type == html.TextNode {
			text := n.Data
			// перед добавлением: нормализация пробелов позже; сейчас точный маппинг
			for i := 0; i < len(text); i++ {
				// добавляем символ в буфер
				buf.WriteByte(text[i])
				indexMap = append(indexMap, htmlPos+i) // позиция в html перед символом
			}
			return htmlPos + len(text)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			htmlPos = walk(c, htmlPos)
		}
		return htmlPos
	}
	// Примерная оценка смещения: для простоты считаем нулём,
	// т.к. мы будем вставлять по indexMap в конкретном htmlBlock,
	// без привязки к глобальному документу.
	walk(node, 0)

	plain := buf.String()
	// теперь нормализуем plain так же, как нормализуем сниппеты
	// но, чтобы не потерять карту, пересоберём карту под нормализованный текст
	normalized := normalizeText(plain)
	// Перестроим карту "как сможем": делаем двухуказательный проход по plain и normalized
	// Схлопывание пробелов может склеить позиции — возьмём первую попавшуюся
	newIndex := make([]int, 0, len(normalized))
	i, j := 0, 0
	for i < len(plain) && j < len(normalized) {
		pc := plain[i]
		nc := normalized[j]
		// приводим регистр?
		if pc == nc {
			newIndex = append(newIndex, indexMap[i])
			i++
			j++
			continue
		}
		// если plain[i] пробел и в normalized уже схлопнуто — пропускаем лишние пробелы
		if unicode.IsSpace(rune(pc)) {
			i++
			continue
		}
		if unicode.IsSpace(rune(nc)) {
			// normalized вставил пробел — попробуем пропустить шум в plain до ближайшего непробельного
			i++
			continue
		}
		// иное несовпадение: сдвигаемся в plain (теги/шумы уже выкинуты)
		i++
	}
	// защитимся от пустоты
	if len(newIndex) == 0 && len(normalized) == 0 {
		newIndex = []int{0}
	}
	return plainIndex{IndexMap: newIndex, Plain: normalized}, nil
}

// токенизация сниппета
func tokens(s string) []string {
	s = stripMarkdown(s)
	s = normalizeText(s)
	re := regexp.MustCompile(`[A-Za-zА-Яа-я0-9]+|[.,:;!?()-]`)
	toks := re.FindAllString(s, -1)
	// выкинем очень короткие/малозначимые токены (кроме пунктуации из правил)
	out := make([]string, 0, len(toks))
	for _, t := range toks {
		if len([]rune(t)) >= 2 || regexp.MustCompile(`[.,:;!?()-]`).MatchString(t) {
			out = append(out, t)
		}
	}
	return out
}

// последовательный поиск токенов с допуском пробелов/шума
type match struct {
	Start, End int
	Score      int
	Found      bool
}

func findBestMatch(snippet string, plain string) match {
	toks := tokens(snippet)
	if len(toks) == 0 || len(plain) == 0 {
		return match{}
	}
	// простая эвристика: строим регекс вида \btok1\b.*?\btok2\b.*?\btok3\b ...
	// (.*? допускает пробелы/шум); но plain уже без тегов, с норм пробелами.
	// Добавим \s+ вместо .*, чтобы не “переедать”
	var b strings.Builder
	b.WriteString(`(?s)`)
	for i, t := range toks {
		if i > 0 {
			b.WriteString(`\s+`)
		}
		b.WriteString(`\Q` + t + `\E`)
	}
	re := regexp.MustCompile(b.String())
	loc := re.FindStringIndex(plain)
	if loc != nil {
		return match{Start: loc[0], End: loc[1], Score: loc[1] - loc[0], Found: true}
	}
	// fallback: попробуем обрезать крайние токены
	for cut := 1; cut <= 2 && len(toks)-cut >= 2; cut++ {
		sub := strings.Join(toks[0:len(toks)-cut], " ")
		re2 := regexp.MustCompile(`(?s)\Q` + sub + `\E`)
		loc2 := re2.FindStringIndex(plain)
		if loc2 != nil {
			return match{Start: loc2[0], End: loc2[1], Score: loc2[1] - loc2[0], Found: true}
		}
	}
	return match{}
}

// оборачивание диапазона в HTML (по байтовым индексам)
func wrapHTMLSpan(htmlBlock string, startByte, endByte int, spanID string) (string, error) {
	if startByte < 0 || endByte > len(htmlBlock) || startByte >= endByte {
		return htmlBlock, fmt.Errorf("bad range")
	}
	return htmlBlock[:startByte] + `<span data-error="` + spanID + `">` +
		htmlBlock[startByte:endByte] + `</span>` + htmlBlock[endByte:], nil
}

//// readLinesFromString разбирает переданный Markdown-текст на строки.
//// Возвращает срез строк и ошибку, если она возникла при сканировании (очень маловероятна для строки).
//func readLinesFromString(md string) ([]string, error) {
//	scanner := bufio.NewScanner(strings.NewReader(md))
//	var lines []string
//	for scanner.Scan() {
//		lines = append(lines, scanner.Text())
//	}
//	if err := scanner.Err(); err != nil {
//		return nil, err
//	}
//	return lines, nil
//}
//
//// extractAndNormalize извлекает текст из HTML и нормализует его
//func extractAndNormalize(htmlStr string) string {
//	var sb strings.Builder
//	doc, err := html.Parse(strings.NewReader(htmlStr))
//	if err != nil {
//		return ""
//	}
//	var f func(*html.Node)
//	f = func(n *html.Node) {
//		if n.Type == html.TextNode {
//			sb.WriteString(n.Data)
//		}
//		for c := n.FirstChild; c != nil; c = c.NextSibling {
//			f(c)
//		}
//	}
//	f(doc)
//	return normalize(sb.String())
//}
//
//// normalize убирает Markdown-разметку, пунктуацию и приводит к нижнему регистру
//func normalize(s string) string {
//	reMd := regexp.MustCompile(`\*\*|__|\[|\]|\([^)]*\)`)
//	s = reMd.ReplaceAllString(s, "")
//	s = strings.ToLower(s)
//	s = strings.Trim(s, ` ,.-–—!?:;"'`)
//	reSp := regexp.MustCompile(`\s+`)
//	s = reSp.ReplaceAllString(s, " ")
//	return strings.TrimSpace(s)
//}
//
//// HighlightPhraseIgnoreCase ищет фразу без учета регистра в указанном блоке
//func HighlightPhraseIgnoreCase(text, phrase string, id int, blockNum string) string {
//	if phrase == "" || blockNum == "" {
//		return text
//	}
//
//	// Ищем блок с указанным номером
//	blockPattern := fmt.Sprintf(`<[^>]*\b%s\b[^>]*>.*?</[^>]*>`, regexp.QuoteMeta(blockNum))
//	blockRegex := regexp.MustCompile(blockPattern)
//
//	// Находим блок
//	blockMatch := blockRegex.FindString(text)
//	if blockMatch == "" {
//		return text // Блок не найден
//	}
//
//	blockStart := strings.Index(text, blockMatch)
//	if blockStart == -1 {
//		return text
//	}
//
//	lowerBlockContent := strings.ToLower(blockMatch)
//	lowerPhrase := strings.ToLower(phrase)
//
//	// Ищем фразу только в содержимом блока
//	index := strings.Index(lowerBlockContent, lowerPhrase)
//	if index == -1 {
//		return text // Фраза не найдена в блоке
//	}
//
//	modifiedBlock := blockMatch
//
//	// Заменяем все вхождения фразы в блоке
//	for index != -1 {
//		// Извлекаем оригинальную фразу с сохранением регистра
//		originalPhrase := modifiedBlock[index : index+len(phrase)]
//		escapedPhrase := html.EscapeString(originalPhrase)
//		highlightedPhrase := fmt.Sprintf(`<span error-id="%d">%s</span>`, id, escapedPhrase)
//
//		// Заменяем найденную фразу в блоке
//		modifiedBlock = modifiedBlock[:index] + highlightedPhrase + modifiedBlock[index+len(phrase):]
//
//		// Ищем следующее вхождение
//		searchStart := index + len(highlightedPhrase)
//		if searchStart >= len(modifiedBlock) {
//			break
//		}
//
//		lowerModifiedBlock := strings.ToLower(modifiedBlock[searchStart:])
//		nextIndex := strings.Index(lowerModifiedBlock, lowerPhrase)
//		if nextIndex == -1 {
//			break
//		}
//		index = searchStart + nextIndex
//	}
//
//	// Заменяем оригинальный блок на модифицированный в полном тексте
//	result := strings.Replace(text, blockMatch, modifiedBlock, 1)
//
//	return result
//}
//
//// fuzzyFind ищет ближайшее вхождение по Левенштейну и возвращает индекс и расстояние
//func fuzzyFind(text, pat string) (int, int) {
//	minDist := len(pat)
//	minIdx := -1
//	for i := 0; i+len(pat) <= len(text); i++ {
//		segment := text[i : i+len(pat)]
//		dist := levenshtein.ComputeDistance(segment, pat)
//		if dist < minDist {
//			minDist = dist
//			minIdx = i
//		}
//		if dist == 0 {
//			break
//		}
//	}
//	return minIdx, minDist
//}
//
//// injectSpan оборачивает найденный HTML-фрагмент в span с data-error
//func injectSpan(htmlStr, normSnippet string, normIdx int, errID string) string {
//	if normSnippet == "" {
//		return htmlStr
//	}
//
//	spanStart := fmt.Sprintf(`<span data-error="%s">`, errID)
//	spanEnd := `</span>`
//	return strings.Replace(htmlStr, normSnippet, spanStart+normSnippet+spanEnd, 1)
//}
//
//func getInt(p *int) int {
//	if p == nil {
//		return 0
//	}
//	return *p
//}
//
//// extractOriginalSnippet пытается сопоставить позицию normIdx
//// в blk.NormText с соответствующим куском в blk.HtmlContent.
//// Очень упрощённый вариант: последовательно удаляем теги из html,
//// но запоминаем границы для реконструкции оригинала.
//func extractOriginalSnippet(htmlStr, normText, normSnippet string, normIdx int) string {
//	// Убираем теги, но при этом запоминаем срезы:
//	type seg struct{ text, html string }
//	var segs []seg
//	var bufTxt, bufHtml strings.Builder
//	inTag := false
//	for _, r := range htmlStr {
//		if r == '<' {
//			inTag = true
//			if bufTxt.Len() > 0 {
//				segs = append(segs, seg{bufTxt.String(), bufHtml.String()})
//				bufTxt.Reset()
//				bufHtml.Reset()
//			}
//			bufHtml.WriteRune(r)
//		} else if r == '>' {
//			bufHtml.WriteRune(r)
//			inTag = false
//		} else {
//			bufHtml.WriteRune(r)
//			if !inTag {
//				bufTxt.WriteRune(r)
//			}
//		}
//	}
//	if bufTxt.Len() > 0 {
//		segs = append(segs, seg{bufTxt.String(), bufHtml.String()})
//	}
//	// Теперь проходим по сегментам, копим normText и ищем normIdx
//	acc := 0
//	for _, s := range segs {
//		if acc+len(s.text) < normIdx {
//			acc += len(s.text)
//			continue
//		}
//		// нужный фрагмент начинается в этом сегменте
//		rel := normIdx - acc
//		if rel+len(normSnippet) <= len(s.text) {
//			// оригинал — такой же кусок из html
//			return s.html[rel : rel+len(normSnippet)]
//		}
//		break
//	}
//	return ""
//}
//
//// injectSpanRaw оборачивает именно найденный оригинальный кусок
//func injectSpanRaw(htmlStr, original, errID string) string {
//	start := fmt.Sprintf(`<span data-error="%s">`, errID)
//	end := `</span>`
//	return strings.Replace(htmlStr, original, start+original+end, 1)
//}
//
//// wrapSnippetInSpan парсит htmlStr, находит в текстовых узлах фразу snippet
//// и оборачивает её в <span data-error="errID">…</span>.
//// Возвращает новый HTML или ошибку.
//func wrapSnippetInSpan(htmlStr, snippet, errID string) (string, error) {
//	// 1) Парсим HTML в дерево
//	doc, err := html.Parse(strings.NewReader(htmlStr))
//	if err != nil {
//		return "", fmt.Errorf("html.Parse: %w", err)
//	}
//
//	// 2) Рекурсивно обходим дерево
//	var f func(*html.Node)
//	f = func(n *html.Node) {
//		// Если это текстовый узел и он содержит наш сниппет
//		if n.Type == html.TextNode {
//			idx := strings.Index(n.Data, snippet)
//			if idx >= 0 {
//				// Разбиваем текст на до-, совпадение и после-части
//				before := n.Data[:idx]
//				match := n.Data[idx : idx+len(snippet)]
//				after := n.Data[idx+len(snippet):]
//
//				// Создаём span-узел
//				span := &html.Node{
//					Type: html.ElementNode,
//					Data: "span",
//					Attr: []html.Attribute{
//						{Key: "error-id", Val: errID},
//					},
//				}
//				span.AppendChild(&html.Node{Type: html.TextNode, Data: match})
//
//				// Вставляем: beforeTextNode, span, afterTextNode вместо оригинального n
//				parent := n.Parent
//				parent.InsertBefore(&html.Node{Type: html.TextNode, Data: before}, n)
//				parent.InsertBefore(span, n)
//				parent.InsertBefore(&html.Node{Type: html.TextNode, Data: after}, n)
//				parent.RemoveChild(n)
//
//				// Прекращаем рекурсию для этого узла — snippet только один раз
//				return
//			}
//		}
//		// Иначе идём глубже
//		for c := n.FirstChild; c != nil; c = c.NextSibling {
//			f(c)
//		}
//	}
//	f(doc)
//
//	// 3) Рендерим обратно в строку
//	var buf bytes.Buffer
//	if err := html.Render(&buf, doc); err != nil {
//		return "", fmt.Errorf("html.Render: %w", err)
//	}
//	return buf.String(), nil
//}

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
