package tzservice

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

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
) (finalHTML string, out []OutError, reportStr string, err error) {
	// Подготовка: собираем рабочие блоки по mapping-ам
	type blockWork struct {
		Map   markdown_service_client.Mapping
		Plain plainIndex // { PlainOrig, PlainNorm }
		HTML  string     // outerHTML блока (либо листа, если это подблок)
	}

	blocks := make(map[string]*blockWork, len(mappings))
	for i := range mappings {
		m := mappings[i]
		pi, perr := buildPlainAndIndexFromHTML(m.HtmlContent)
		if perr != nil {
			if log != nil {
				log.Warn("buildPlainAndIndexFromHTML failed", "element", m.ElementID, "err", perr)
			}
			// всё равно добавим блок, чтобы мы могли его тупо заменить позже, если понадобится
			blocks[m.ElementID] = &blockWork{Map: m, Plain: plainIndex{PlainOrig: "", PlainNorm: ""}, HTML: m.HtmlContent}
			continue
		}
		blocks[m.ElementID] = &blockWork{Map: m, Plain: pi, HTML: m.HtmlContent}
	}

	// Для отчёта: строки TSV
	var reportLines []string
	reportLines = append(reportLines, "error_id err_type code group line_start line_end candidates chosen_element status reason snippet snippet_norm candidates_preview")

	// небольшие утилиты для отчёта
	toStr := func(p *int) string {
		if p == nil {
			return ""
		}
		return strconv.Itoa(*p)
	}
	joinIDs := func(xs []*blockWork) string {
		if len(xs) == 0 {
			return ""
		}
		ids := make([]string, 0, len(xs))
		for _, b := range xs {
			ids = append(ids, b.Map.ElementID)
		}
		return strings.Join(ids, ",")
	}
	addReport := func(id string, inst ErrorInstance, candidates []*blockWork, chosen string, status string, reason string) {
		normSnippet := normalizeText(stripMarkdown(inst.Snippet))
		if len([]rune(normSnippet)) > 180 {
			normSnippet = string([]rune(normSnippet)[:180]) + "…"
		}

		candPrev := ""
		if len(candidates) > 0 {
			previews := make([]string, 0, len(candidates))
			for _, c := range candidates {
				prev := c.Plain.PlainNorm
				if len([]rune(prev)) > 80 {
					prev = string([]rune(prev)[:80]) + "…"
				}
				previews = append(previews, c.Map.ElementID+"::"+prev)
			}
			candPrev = strings.Join(previews, " | ")
		}

		// snippet без перевода строк и усечённый
		sn := strings.TrimSpace(inst.Snippet)
		rs := []rune(sn)
		if len(rs) > 180 {
			sn = string(rs[:180]) + "…"
		}
		line := strings.Join([]string{
			id, inst.ErrType, inst.Code, inst.GroupID,
			toStr(inst.LineStart), toStr(inst.LineEnd),
			joinIDs(candidates), chosen, status, reason, sn, normSnippet, candPrev,
		}, " ")
		reportLines = append(reportLines, line)
	}

	errCounter := 0
	out = make([]OutError, 0, len(errors))

	for _, inst := range errors {
		rid := fmt.Sprintf("e-%03d", errCounter)
		errCounter++

		// "missing" — только в список ошибок и в отчёт
		if strings.ToLower(inst.ErrType) == "missing" {
			out = append(out, OutError{
				ID:           rid,
				GroupID:      inst.GroupID,
				Code:         inst.Code,
				SuggestedFix: inst.SuggestedFix,
				Rationale:    inst.Rationale,
			})
			addReport(rid, inst, nil, "", "skipped", "missing: общий комментарий, без вставки в HTML")
			continue
		}

		if strings.ToLower(inst.ErrType) != "invalid" {
			// непредусмотренный тип — пропускаем
			addReport(rid, inst, nil, "", "skipped", "unsupported err_type")
			continue
		}

		// 1) кандидаты по линиям
		var candidates []*blockWork
		if inst.LineStart != nil && inst.LineEnd != nil {
			for _, bl := range blocks {
				if !(bl.Map.MarkdownEnd < *inst.LineStart || bl.Map.MarkdownStart > *inst.LineEnd) {
					candidates = append(candidates, bl)
				}
			}
		}
		if len(candidates) == 0 {
			// если не нашли по линиям — ищем по всем
			for _, bl := range blocks {
				candidates = append(candidates, bl)
			}
		}

		// 2) Поиск лучшего совпадения:
		//    - если блок — контейнер (ul/ol/div/table/...), спускаемся в листья и ищем в каждом;
		//    - выигрывает тот, у кого покрытие (end-start) больше.
		type pick struct {
			host *blockWork // исходный mapping-блок
			leaf *blockWork // листовой фрагмент (может совпадать с host)
			m    match      // {Start, End, Found} — индексы по leaf.Plain.PlainNorm
		}
		var best pick

		for _, host := range candidates {
			targets := []*blockWork{host}
			top := detectTopTagName(host.HTML)
			if isContainer(top) {
				leaves, lerr := collectLeafFragments(host.HTML)
				if lerr == nil && len(leaves) > 0 {
					targets = targets[:0]
					for _, leafHTML := range leaves {
						pi, perr := buildPlainAndIndexFromHTML(leafHTML)
						if perr != nil {
							continue
						}
						targets = append(targets, &blockWork{
							Map: markdown_service_client.Mapping{
								ElementID:     host.Map.ElementID,
								HtmlContent:   leafHTML,
								MarkdownStart: host.Map.MarkdownStart,
								MarkdownEnd:   host.Map.MarkdownEnd,
							},
							Plain: pi,
							HTML:  leafHTML,
						})
					}
				}
			}

			for _, leaf := range targets {
				if st, en, ok := quickSubstringMatch(inst.Snippet, leaf.Plain.PlainNorm); ok {
					m := match{Start: st, End: en, Found: true}
					if best.host == nil || (m.End-m.Start) > (best.m.End-best.m.Start) {
						best = pick{host: host, leaf: leaf, m: m}
					}
					continue
				}

				m := findBestMatch(inst.Snippet, leaf.Plain.PlainNorm)
				if m.Found {
					// выбираем по максимальному покрытию
					if best.host == nil || (m.End-m.Start) > (best.m.End-best.m.Start) {
						best = pick{host: host, leaf: leaf, m: m}
					}
				}
			}
		}

		// В список ошибок добавляем ВСЕ invalid (даже если не нашли место вставки),
		// чтобы фронт видел весь набор.
		out = append(out, OutError{
			ID:           rid,
			GroupID:      inst.GroupID,
			Code:         inst.Code,
			SuggestedFix: inst.SuggestedFix,
			Rationale:    inst.Rationale,
		})

		if best.host == nil {
			// не нашли совпадение
			addReport(rid, inst, candidates, "", "not-found", "совпадение не найдено ни в одном кандидате")
			if log != nil {
				log.Warn("invalid not matched", "id", rid, "code", inst.Code, "group", inst.GroupID, "snippet", inst.Snippet)
			}
			continue
		}

		// 3) Проецируем диапазон из PlainNorm → PlainOrig, затем оборачиваем внутри leaf.HTML
		startOrig, endOrig := mapNormToOrig(best.leaf.Plain.PlainOrig, best.leaf.Plain.PlainNorm, best.m.Start, best.m.End)
		wrapped, werr := wrapInLeafHTML(best.leaf.HTML, best.leaf.Plain.PlainOrig, startOrig, endOrig, rid)
		if werr != nil {
			addReport(rid, inst, candidates, best.host.Map.ElementID, "skipped", "wrap failed: "+werr.Error())
			if log != nil {
				log.Warn("wrap failed", "id", rid, "element", best.host.Map.ElementID, "err", werr)
			}
			continue
		}

		// 4) Возвращаем лист в хост и пересобираем plain хоста (для последующих вставок)
		if best.leaf != best.host {
			best.host.HTML = strings.Replace(best.host.HTML, best.leaf.HTML, wrapped, 1)
		} else {
			best.host.HTML = wrapped
		}
		if pi, perr := buildPlainAndIndexFromHTML(best.host.HTML); perr == nil {
			best.host.Plain = pi
		}

		addReport(rid, inst, candidates, best.host.Map.ElementID, "found", "")
		if log != nil {
			log.Info("invalid matched",
				"id", rid,
				"element", best.host.Map.ElementID,
				"normStart", best.m.Start,
				"normEnd", best.m.End,
				"startOrig", startOrig,
				"endOrig", endOrig,
			)
		}
	}

	// 5) Пересборка итогового HTML: заменяем data-mapping-id соответствующим outerHTML блока
	doc, derr := goquery.NewDocumentFromReader(strings.NewReader(htmlWithIds))
	if derr != nil {
		err = derr
		return htmlWithIds, out, strings.Join(reportLines, "\n"), err
	}

	for _, b := range blocks {
		sel := fmt.Sprintf(`[data-mapping-id="%s"]`, b.Map.ElementID)
		doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
			// Заменяем весь узел на наш HTML (outerHTML)
			_ = s.ReplaceWithHtml(b.HTML)
		})
	}

	htmlRes, herr := doc.Html()
	if herr != nil {
		err = herr
		return htmlWithIds, out, strings.Join(reportLines, "\n"), err
	}

	finalHTML = htmlRes
	reportStr = strings.Join(reportLines, "\n")
	return
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
//func normalizeText(s string) string {
//	// 1) HTML entities -> символы
//	s = html.UnescapeString(s)
//	// 2) заменить NBSP и прочие хитрые пробелы на обычный
//	s = strings.ReplaceAll(s, "\u00A0", " ")
//	s = strings.ReplaceAll(s, "\u2009", " ")
//	s = strings.ReplaceAll(s, "\u202F", " ")
//	// 3) умные кавычки/тире -> обычные
//	repl := map[string]string{
//		"“": "\"", "”": "\"", "«": "\"", "»": "\"", "‘": "'", "’": "'",
//		"—": "-", "–": "-",
//	}
//	for k, v := range repl {
//		s = strings.ReplaceAll(s, k, v)
//	}
//	// 4) схлопываем пробелы
//	s = strings.TrimSpace(s)
//	reSpaces := regexp.MustCompile(`\s+`)
//	s = reSpaces.ReplaceAllString(s, " ")
//	return s
//}

//func stripMarkdown(s string) string {
//	// очень лёгкая чистка markdown (для сниппетов):
//	// **bold**, *i*, __u__, `code`, [text](url), начальные "- " / "* " списков
//	s = regexp.MustCompile("`([^`]*)`").ReplaceAllString(s, "$1")
//	s = strings.NewReplacer("**", "", "__", "", "*", "", "_", "").Replace(s)
//	s = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(s, "$1")
//	s = regexp.MustCompile(`(?m)^\s*[-*]\s+`).ReplaceAllString(s, "")
//	return s
//}

type plainIndex struct {
	IndexMap  []int  // по PlainOrig: позиция байта в HTML перед символом PlainOrig[i]
	PlainOrig string // сырой текст без тегов, пробелы можно схлопнуть
	PlainNorm string // нормализованный текст для поиска
}

// аккуратно вытаскиваем именно текст-ноды и строим позиционную карту
func buildPlainAndIndexFromHTML(htmlBlock string) (plainIndex, error) {
	n, err := html.Parse(strings.NewReader(htmlBlock))
	if err != nil {
		return plainIndex{}, err
	}

	// Пройдёмся по дереву и соберём текст без тегов + карту позиций в исходном htmlBlock.
	// ВНИМАНИЕ: парасер строит новое дерево и у нас нет «реальных» смещений.
	// Поэтому пойдём проще: удалим теги вручную на копии строки, но это ненадёжно.
	// Надёжнее: рендерить обратно ВНУТРЕННОСТИ узлов текста мы не сможем.
	// Пойдём практично: конкатенируем TextNode.Data, карта будет «виртуальная» (по тексту),
	// а для wrap будем находить подстроку в htmlBlock через text и локальный поиск. Это работает,
	// если внутри одного “листа” текстовая последовательность не размазана сложными нодами.
	// Для устойчивости оставим IndexMap как «индекс в htmlBlock по локальному поиску» при вставке.

	// => Проще: для вставки ЛИСТОВЫХ узлов мы заменяем leaf.outerHTML целиком (как уже делаем),
	// а внутри leaf подстроку режем по его собственному HTML (локальный wrap). Для этого
	// IndexMap нам не нужен — нам нужен offset в leaf.HTML. Его мы получим через поиск
	// по PlainOrig внутри leaf.HTML (локальный re-матч текста).
	// Поэтому здесь храним только PlainOrig и PlainNorm.

	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	// ищем первый элемент под body, если это фрагмент
	body := findBody(n)
	if body == nil {
		body = n
	}
	walk(body)

	plainOrig := squeezeSpaces(b.String()) // схлопнем пробелы, но без смены регистра/символов
	plainNorm := normalizeText(plainOrig)  // полноценная нормализация — только для поиска

	return plainIndex{
		IndexMap:  nil, // больше не используем
		PlainOrig: plainOrig,
		PlainNorm: plainNorm,
	}, nil
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

//// последовательный поиск токенов с допуском пробелов/шума
//type match struct {
//	Start, End int
//	Score      int
//	Found      bool
//}

// оборачивание диапазона в HTML (по байтовым индексам)
func wrapHTMLSpan(htmlBlock string, startByte, endByte int, spanID string) (string, error) {
	if startByte < 0 || endByte > len(htmlBlock) || startByte >= endByte {
		return htmlBlock, fmt.Errorf("bad range")
	}
	return htmlBlock[:startByte] + `<span error-id="` + spanID + `">` +
		htmlBlock[startByte:endByte] + `</span>` + htmlBlock[endByte:], nil
}

func squeezeSpaces(s string) string {
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\u2009", " ")
	s = strings.ReplaceAll(s, "\u202F", " ")
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

func findBody(n *html.Node) *html.Node {
	var body *html.Node
	var dfs func(*html.Node)
	dfs = func(x *html.Node) {
		if x.Type == html.ElementNode && x.Data == "body" {
			body = x
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			if body != nil {
				return
			}
			dfs(c)
		}
	}
	dfs(n)
	return body
}
