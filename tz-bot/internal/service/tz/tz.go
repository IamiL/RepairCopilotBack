package tzservice

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
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
	"strings"
	"unicode"
	"unicode/utf8"

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
	var groups []GroupReport
	outHTML, invalidErrors, audit, err := IntegrateErrorsIntoHTML(
		markdownResponse.HtmlWithIds,
		markdownResponse.Mappings, // если типы совпадают — можно напрямую
		groups,
	)
	if err != nil {
		log.Error("Ошибка алгоритма совмещения ошибок с html: ", sl.Err(err))
	}
	// Отправка сообщения с умным делением по границам предложений
	if len(audit) > 3999 {
		messages := tz.splitMessage(audit, 4000)
		for _, msg := range messages {
			tz.tgClient.SendMessage(msg)
		}
	} else {
		tz.tgClient.SendMessage(audit)
	}

	log.Info("аудит: ")
	log.Info(audit)

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

	return outHTML, *css, "123", outInvalidErrors, missingErrorsResponse, "123", nil
}

type AuditLine struct {
	GroupID     string
	Code        string
	ErrType     string
	SnippetRaw  string
	SnippetNorm string
	LineStart   *int
	LineEnd     *int
	ElementID   string
	HtmlTag     string
	Status      string // FOUND | NOT_FOUND
	Note        string // позиции/детали или причина
}

type textRun struct {
	Node   *html.Node
	Offset int // byte offset внутри Node.Data
	Len    int // bytes
}

type concatIndex struct {
	NormText string
	Runs     []textRun // по порядку, покрывают NormText
	// map: глобальный offset в NormText -> (runIdx, localOffset)
	// строим на лету в mapIndex()
	posToRun []struct{ runIdx, localOff int }
}

// ==== Нормализация: markdown/HTML → «сопоставимый текст» ====

var mdInlineREs = []*regexp.Regexp{
	regexp.MustCompile(`\*\*(.*?)\*\*`),
	regexp.MustCompile(`__(.*?)__`),
	regexp.MustCompile("`([^`]*)`"),
	regexp.MustCompile(`\[(.*?)\]\((.*?)\)`), // [text](url) -> text
}

func normalizeSnippet(s string) string {
	x := s
	// Снимем markdown-инлайн
	for _, re := range mdInlineREs {
		x = re.ReplaceAllString(x, `$1`)
	}
	// HTML entities → текст (на всякий)
	x = html.UnescapeString(x)
	// Ё->Е, кавычки/дефисы → базовые, убрать лишнюю пунктуацию (кроме букв/цифр/пробелов)
	x = unifyRunes(x)
	// Схлопнем пробелы
	x = collapseSpaces(x)
	return x
}

func unifyRunes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '«', '»', '“', '”', '„', '‟', '″', '＂':
			r = '"'
		case '’', '‘', '‚', '′', '＇':
			r = '\''
		case '–', '—', '−', '-':
			r = '-' // минусы/дефисы
		case 'ё':
			r = 'е'
		case 'Ё':
			r = 'Е'
		}
		// Оставим буквы/цифры/пробелы/основную пунктуацию .,:;!?-'"()/ — остальное уберём
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) ||
			strings.ContainsRune(`.,:;!?-"'/()`, r) {
			b.WriteRune(r)
		}
		// прочее — пропускаем (снимаем визуальные артефакты)
	}
	return b.String()
}

func collapseSpaces(s string) string {
	// Заменим любые пробельные на одиночный пробел
	re := regexp.MustCompile(`\s+`)
	out := re.ReplaceAllString(strings.TrimSpace(s), " ")
	// Снимем пробел перед точкой/запятой/… (частый артефакт)
	out = regexp.MustCompile(`\s+([.,:;!?])`).ReplaceAllString(out, "$1")
	return out
}

// ==== Построение индекса по HTML: собираем нормализованный «плоский» текст и карту смещений ====

func buildConcatIndexFromHTML(htmlFrag string) (*concatIndex, *html.Node, error) {
	root, err := html.Parse(strings.NewReader(htmlFrag))
	if err != nil {
		return nil, nil, fmt.Errorf("parse html: %w", err)
	}

	var runs []textRun
	var buf strings.Builder
	// Повторим нормализацию для html-текста (по текстовым узлам)
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			orig := n.Data
			if strings.TrimSpace(orig) != "" {
				// Нормализуем текст узла в ту же систему координат
				norm := normalizeSnippet(orig)
				if norm != "" {
					offset := len(buf.String())
					buf.WriteString(norm)
					runs = append(runs, textRun{Node: n, Offset: 0, Len: len(n.Data)}) // Offset/Len в исходном тексте (byte). Корректируем при вставке.
					_ = offset
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	ci := &concatIndex{
		NormText: buf.String(),
		Runs:     runs,
	}
	ci.mapIndex()
	return ci, root, nil
}

// Грубая, но эффективная карта: каждый rune в NormText → индекс run + локальное смещение
func (ci *concatIndex) mapIndex() {
	ci.posToRun = make([]struct{ runIdx, localOff int }, 0, utf8.RuneCountInString(ci.NormText))
	//var (
	//	curRun = 0
	//)
	// Упрощённо: считаем, что каждый run добавлялся целиком norm-текстом узла.
	// Для «сдвигов» внутри узла этого достаточно, т.к. мы всегда отматываем по порядку.
	//runes := []rune(ci.NormText)
	//for i := range runes {
	//	// Найти текущий run по доле длины (приблизительно). Так как мы не держим отдельные длины норм-узлов,
	//	// пойдём от начала: равномерно распределять нельзя, поэтому проще хранить границы.
	//	// Упростим: разобьём NormText на равные куски последовательно по числу runs.
	//	// Для точности лучше хранить per-run длину norm-строк. Давайте её посчитаем.
	//}
	// Переделаем: посчитаем норм-строки по-узлово отдельно
}

func buildConcatIndexFromHTMLPrecise(htmlFrag string) (*concatIndex, *html.Node, error) {
	root, err := html.Parse(strings.NewReader(htmlFrag))
	if err != nil {
		return nil, nil, fmt.Errorf("parse html: %w", err)
	}
	type nodeChunk struct {
		runIdx int
		norm   string
	}
	var runs []textRun
	var chunks []nodeChunk
	var flat strings.Builder

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			orig := n.Data
			if strings.TrimSpace(orig) != "" {
				norm := normalizeSnippet(orig)
				if norm != "" {
					runs = append(runs, textRun{Node: n, Offset: 0, Len: len(orig)})
					chunks = append(chunks, nodeChunk{runIdx: len(runs) - 1, norm: norm})
					flat.WriteString(norm)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	ci := &concatIndex{
		NormText: flat.String(),
		Runs:     runs,
	}
	// Построим posToRun точно
	var posMap []struct{ runIdx, localOff int }
	posMap = make([]struct{ runIdx, localOff int }, 0, len([]rune(ci.NormText)))
	pos := 0
	for _, ch := range chunks {
		runes := []rune(ch.norm)
		for i := 0; i < len(runes); i++ {
			posMap = append(posMap, struct{ runIdx, localOff int }{runIdx: ch.runIdx, localOff: i})
			pos++
		}
	}
	ci.posToRun = posMap
	return ci, root, nil
}

// ==== Поиск нормализованного сниппета в нормализованном тексте блока ====

func findNormalized(haystack, needle string) (startRune, endRune int, ok bool) {
	if needle == "" || haystack == "" {
		return 0, 0, false
	}
	// Простой индекс по рунам
	H := []rune(haystack)
	N := []rune(needle)
	HL := len(H)
	NL := len(N)
	if NL > HL {
		return 0, 0, false
	}
	// На больших текстах имеет смысл KMP/2-gram индекс; тут — простой проход
	for i := 0; i <= HL-NL; i++ {
		match := true
		for j := 0; j < NL; j++ {
			if H[i+j] != N[j] {
				match = false
				break
			}
		}
		if match {
			return i, i + NL, true
		}
	}
	return 0, 0, false
}

// ==== Оборачивание совпадения в <span data-error="..."> внутри DOM ====

func wrapMatchInDOM(root *html.Node, ci *concatIndex, startRune, endRune int, spanID string) (bool, string, error) {
	if startRune >= endRune {
		return false, "", nil
	}
	// Преобразуем глобальные rune-позиции в списки (runIdx, localOff)
	if startRune < 0 || endRune > len(ci.posToRun) {
		return false, "", fmt.Errorf("range out of bounds")
	}
	start := ci.posToRun[startRune]
	end := ci.posToRun[endRune-1] // включительно
	// Мы могли задеть несколько textNode-ов: надо оборачивать куски последовательно.
	// Подход: создаём один общий <span>, а внутрь перемещаем фрагменты, которые пересекают этот диапазон —
	// но DOM не позволит положить в один span части, разделённые тегами. Поэтому делаем per-node обёртку.
	// Это ОК: одна ошибка → несколько span'ов с одинаковым data-error.

	// Генерим атрибуты
	// <span data-error="spanID"></span>
	makeSpan := func() *html.Node {
		span := &html.Node{
			Type: html.ElementNode,
			Data: "span",
			Attr: []html.Attribute{{Key: "data-error", Val: spanID}},
		}
		return span
	}

	// Упрощение: оборачиваем каждый задействованный текстовый узел в части его диапазона.
	// Идём от start до end, шагами по posToRun, группируя по runIdx.
	type seg struct{ runIdx, fromLocal, toLocal int } // rune-based
	segs := make([]seg, 0, 4)
	curRun := start.runIdx
	from := start.localOff
	for i := startRune; i < endRune; i++ {
		pr := ci.posToRun[i]
		if pr.runIdx != curRun {
			// закрываем предыдущий сегмент [from, last+1)
			segs = append(segs, seg{runIdx: curRun, fromLocal: from, toLocal: ci.posToRun[i-1].localOff + 1})
			curRun = pr.runIdx
			from = pr.localOff
		}
	}
	// хвост
	segs = append(segs, seg{runIdx: curRun, fromLocal: from, toLocal: end.localOff + 1})

	// Теперь на каждом textNode делим строку (по рунам) и вставляем span вокруг средины
	for _, s := range segs {
		n := ci.Runs[s.runIdx].Node
		orig := n.Data
		runes := []rune(orig)

		// Для корректности нам нужно «сопоставление нормализованных рун → исходные руны».
		// Мы упростили: нормализовали orig → norm и считали позиции по norm.
		// Чтобы точно разрезать исходный текст, можно повторно пройти orig, формируя такую же норм-строку и
		// запоминая соответствие индексов. Сделаем helper:

		rawFrom, rawTo := mapNormalizedSliceToRaw(orig, s.fromLocal, s.toLocal)

		if rawFrom < 0 || rawTo > len(runes) || rawFrom >= rawTo {
			continue // защитимся
		}

		before := string(runes[:rawFrom])
		middle := string(runes[rawFrom:rawTo])
		after := string(runes[rawTo:])

		parent := n.Parent
		if parent == nil {
			continue
		}

		// Создаём узлы: before, <span>middle</span>, after
		var beforeNode *html.Node
		if before != "" {
			beforeNode = &html.Node{Type: html.TextNode, Data: before}
			parent.InsertBefore(beforeNode, n)
		}
		span := makeSpan()
		span.AppendChild(&html.Node{Type: html.TextNode, Data: middle})
		parent.InsertBefore(span, n)
		var afterNode *html.Node
		if after != "" {
			afterNode = &html.Node{Type: html.TextNode, Data: after}
			parent.InsertBefore(afterNode, n)
		}
		parent.RemoveChild(n) // удаляем исходный

	}
	// Сериализуем root обратно в строку
	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return false, "", err
	}
	return true, buf.String(), nil
}

// Сопоставление: локальный отрезок нормализованной строки textNode → диапазон в raw runes
func mapNormalizedSliceToRaw(raw string, normFrom, normTo int) (rawFrom, rawTo int) {
	// Строим нормализованный рун-поток побуквенно, параллельно запоминая «какая raw-руна попала в какой norm-индекс»
	rawRunes := []rune(raw)
	normIndex := 0
	rawIndexAtNorm := make([]int, 0, len(rawRunes))
	for i, r := range rawRunes {
		nr := r
		switch nr {
		case '«', '»', '“', '”', '„', '‟', '″', '＂':
			nr = '"'
		case '’', '‘', '‚', '′', '＇':
			nr = '\''
		case '–', '—', '−', '-':
			nr = '-'
		case 'ё':
			nr = 'е'
		case 'Ё':
			nr = 'Е'
		}
		// Фильтр символов так же, как в unifyRunes:
		if unicode.IsLetter(nr) || unicode.IsDigit(nr) || unicode.IsSpace(nr) ||
			strings.ContainsRune(`.,:;!?-"'/()`, nr) {
			// collapsed spaces/trim — сложнее. Здесь мы только считаем соответствие посимвольно.
			// Это даёт достаточно точности на коротких сниппетах. Для production можно сделать полный pipe.
			rawIndexAtNorm = append(rawIndexAtNorm, i)
			normIndex++
		}
	}
	if normFrom < 0 || normTo > len(rawIndexAtNorm) || normFrom >= normTo {
		return -1, -1
	}
	rawFrom = rawIndexAtNorm[normFrom]
	rawTo = rawIndexAtNorm[normTo-1] + 1
	return rawFrom, rawTo
}

// ==== Склейка всего: интеграция ошибок в HTML ====

type GroupReport struct {
	GroupID string `json:"group_id"`
	Errors  []struct {
		Code      string `json:"code"`
		Instances []struct {
			ErrType      string  `json:"err_type"`
			Snippet      string  `json:"snippet"`
			LineStart    *int    `json:"line_start"`
			LineEnd      *int    `json:"line_end"`
			SuggestedFix *string `json:"suggested_fix"`
			Rationale    string  `json:"rationale"`
		} `json:"instances"`
	} `json:"errors"`
}

// Вход: htmlWithIds целиком, mappings, сглаженный список групп/ошибок от LLM
// Выход: обновленный htmlWithIds, outErrors, auditReport
func IntegrateErrorsIntoHTML(
	htmlWithIds string,
	mappings []markdown_service_client.Mapping,
	groupReports []GroupReport,
) (string, []OutError, string, error) {

	// Индекс маппингов по ElementID
	mapByID := make(map[string]markdown_service_client.Mapping, len(mappings))
	for _, m := range mappings {
		mapByID[m.ElementID] = m
	}

	// Для возможности замены фрагментов — сделаем карту id->обновленный html_content
	updatedFrag := make(map[string]string, len(mappings))

	// Соберём единый список инстансов с метаданными кода/группы
	type flat struct {
		GroupID string
		Code    string
		ErrorInstance
	}
	var all []flat
	for _, gr := range groupReports {
		for _, e := range gr.Errors {
			for _, inst := range e.Instances {
				all = append(all, flat{
					GroupID: gr.GroupID,
					Code:    e.Code,
					ErrorInstance: ErrorInstance{
						GroupID:      gr.GroupID,
						Code:         e.Code,
						ErrType:      inst.ErrType,
						Snippet:      inst.Snippet,
						LineStart:    inst.LineStart,
						LineEnd:      inst.LineEnd,
						SuggestedFix: inst.SuggestedFix,
						Rationale:    inst.Rationale,
					},
				})
			}
		}
	}

	// Аудит
	var audit []AuditLine
	var out []OutError

	// Хелпер: выбрать кандидатов по строкам
	selectCandidates := func(ls, le *int) []markdown_service_client.Mapping {
		if ls == nil || le == nil {
			return mappings
		}
		L := *ls
		R := *le
		var res []markdown_service_client.Mapping
		for _, m := range mappings {
			// Пересечение диапазонов
			if !(m.MarkdownEnd < L || m.MarkdownStart > R) {
				res = append(res, m)
			}
		}
		if len(res) == 0 {
			return mappings // fallback
		}
		return res
	}

	for _, it := range all {
		if it.ErrType == "missing" {
			// Не оборачиваем, просто фиксируем как OutError
			id := makeStableID(it.GroupID, it.Code, it.ErrType, it.Snippet)
			out = append(out, OutError{
				ID:           id,
				GroupID:      it.GroupID,
				Code:         it.Code,
				SuggestedFix: it.SuggestedFix,
				Rationale:    it.Rationale,
			})
			audit = append(audit, AuditLine{
				GroupID:     it.GroupID,
				Code:        it.Code,
				ErrType:     it.ErrType,
				SnippetRaw:  it.Snippet,
				SnippetNorm: normalizeSnippet(it.Snippet),
				LineStart:   it.LineStart,
				LineEnd:     it.LineEnd,
				ElementID:   "",
				HtmlTag:     "",
				Status:      "NOT_APPLICABLE",
				Note:        "missing: не внедряется в HTML",
			})
			continue
		}

		// invalid — ищем и оборачиваем
		normSnippet := normalizeSnippet(it.Snippet)
		if normSnippet == "" {
			// пусто — пропустим
			audit = append(audit, AuditLine{
				GroupID: it.GroupID, Code: it.Code, ErrType: it.ErrType,
				SnippetRaw: it.Snippet, SnippetNorm: normSnippet,
				LineStart: it.LineStart, LineEnd: it.LineEnd,
				Status: "NOT_FOUND", Note: "пустой после нормализации",
			})
			continue
		}

		cands := selectCandidates(it.LineStart, it.LineEnd)
		found := false
		spanID := makeStableID(it.GroupID, it.Code, it.ErrType, it.Snippet)

		for _, cand := range cands {
			frag := cand.HtmlContent
			// Возможно, этот фрагмент уже меняли
			if s, ok := updatedFrag[cand.ElementID]; ok {
				frag = s
			}

			ci, root, err := buildConcatIndexFromHTMLPrecise(frag)
			if err != nil {
				audit = append(audit, AuditLine{
					GroupID: it.GroupID, Code: it.Code, ErrType: it.ErrType,
					SnippetRaw: it.Snippet, SnippetNorm: normSnippet,
					LineStart: it.LineStart, LineEnd: it.LineEnd,
					ElementID: cand.ElementID, HtmlTag: cand.HtmlTag,
					Status: "NOT_FOUND",
					Note:   fmt.Sprintf("parse error: %v", err),
				})
				continue
			}

			start, end, ok := findNormalized(ci.NormText, normSnippet)
			if !ok {
				audit = append(audit, AuditLine{
					GroupID: it.GroupID, Code: it.Code, ErrType: it.ErrType,
					SnippetRaw: it.Snippet, SnippetNorm: normSnippet,
					LineStart: it.LineStart, LineEnd: it.LineEnd,
					ElementID: cand.ElementID, HtmlTag: cand.HtmlTag,
					Status: "NOT_FOUND", Note: "no match in candidate",
				})
				continue
			}

			ok2, newFrag, err := wrapMatchInDOM(root, ci, start, end, spanID)
			if err != nil || !ok2 {
				audit = append(audit, AuditLine{
					GroupID: it.GroupID, Code: it.Code, ErrType: it.ErrType,
					SnippetRaw: it.Snippet, SnippetNorm: normSnippet,
					LineStart: it.LineStart, LineEnd: it.LineEnd,
					ElementID: cand.ElementID, HtmlTag: cand.HtmlTag,
					Status: "NOT_FOUND",
					Note:   fmt.Sprintf("wrap error: %v", err),
				})
				continue
			}

			updatedFrag[cand.ElementID] = newFrag
			found = true
			out = append(out, OutError{
				ID:           spanID,
				GroupID:      it.GroupID,
				Code:         it.Code,
				SuggestedFix: it.SuggestedFix,
				Rationale:    it.Rationale,
			})

			audit = append(audit, AuditLine{
				GroupID: it.GroupID, Code: it.Code, ErrType: it.ErrType,
				SnippetRaw: it.Snippet, SnippetNorm: normSnippet,
				LineStart: it.LineStart, LineEnd: it.LineEnd,
				ElementID: cand.ElementID, HtmlTag: cand.HtmlTag,
				Status: "FOUND",
				Note:   fmt.Sprintf("match [%d..%d] in normalized text", start, end),
			})
			break // нашли в подходящем фрагменте — хватит
		}

		if !found {
			// fallback: поиск по всем, если был узкий диапазон
			if it.LineStart != nil {
				it.LineStart = nil
				it.LineEnd = nil
				// рекурсивно можно, но чтобы не усложнять — просто логируем, что не нашли.
				audit = append(audit, AuditLine{
					GroupID: it.GroupID, Code: it.Code, ErrType: it.ErrType,
					SnippetRaw: it.Snippet, SnippetNorm: normSnippet,
					Status: "NOT_FOUND",
					Note:   "не найдено в указанных строках; расширение поиска по всему документу можно включить флагом",
				})
			}
		}
	}

	// Сборка финального HtmlWithIds: заменим обновлённые фрагменты по data-mapping-id
	finalHTML := applyUpdatedFragments(htmlWithIds, updatedFrag)

	// Аудит → строка (или JSON)
	auditStr := buildAuditReport(audit)
	return finalHTML, out, auditStr, nil
}

func makeStableID(group, code, errType, snippet string) string {
	sum := sha1.Sum([]byte(group + "|" + code + "|" + errType + "|" + snippet))
	return hex.EncodeToString(sum[:8]) // короткий хэш
}

func applyUpdatedFragments(htmlWithIds string, updated map[string]string) string {
	// Каждый фрагмент окружён контейнером с data-mapping-id="ElementID" — мы можем заменить его innerHTML.
	// Проще всего распарсить весь htmlWithIds, пройти по узлам с атрибутом data-mapping-id и, если есть updated[id], заменить их children.
	root, err := html.Parse(strings.NewReader(htmlWithIds))
	if err != nil {
		return htmlWithIds // безопасный fallback
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for i := range n.Attr {
				if n.Attr[i].Key == "data-mapping-id" {
					id := n.Attr[i].Val
					if repl, ok := updated[id]; ok {
						// Заменим детей n на детей из repl
						newNode, err := html.Parse(strings.NewReader(repl))
						if err == nil {
							// newNode это корень документа с <html><head/><body>..., достанем body->firstChild
							body := findFirst(newNode, func(x *html.Node) bool { return x.Type == html.ElementNode && x.Data == "body" })
							if body != nil {
								// Очистим n.Children
								for c := n.FirstChild; c != nil; {
									next := c.NextSibling
									n.RemoveChild(c)
									c = next
								}
								// Перенесём детей body в n
								for c := body.FirstChild; c != nil; c = c.NextSibling {
									n.AppendChild(cloneShallowTree(c))
								}
							}
						}
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	var buf bytes.Buffer
	_ = html.Render(&buf, root)
	return buf.String()
}

func findFirst(n *html.Node, pred func(*html.Node) bool) *html.Node {
	if pred(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if r := findFirst(c, pred); r != nil {
			return r
		}
	}
	return nil
}

func cloneShallowTree(n *html.Node) *html.Node {
	cp := &html.Node{
		Type:     n.Type,
		Data:     n.Data,
		DataAtom: n.DataAtom,
		Attr:     append([]html.Attribute(nil), n.Attr...),
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		cp.AppendChild(cloneShallowTree(c))
	}
	return cp
}

func buildAuditReport(lines []AuditLine) string {
	var b strings.Builder
	for _, a := range lines {
		fmt.Fprintf(&b, "group=%s code=%s type=%s snippet=%q norm=%q lines=[%v..%v] element=%s tag=%s status=%s note=%s\n",
			a.GroupID, a.Code, a.ErrType, a.SnippetRaw, a.SnippetNorm,
			ptrInt(a.LineStart), ptrInt(a.LineEnd), a.ElementID, a.HtmlTag, a.Status, a.Note,
		)
	}
	return b.String()
}

func ptrInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
