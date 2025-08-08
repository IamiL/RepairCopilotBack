package tzservice

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"sort"
	"strings"
	"unicode"

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

	var report []ReportEntry
	errCounter := 0

	for _, inst := range errors {
		rid := fmt.Sprintf("e-%03d", errCounter)
		errCounter++

		rep := ReportEntry{
			ErrorID: rid, GroupID: inst.GroupID, Code: inst.Code,
			ErrType: inst.ErrType, Snippet: inst.Snippet,
			LineStart: inst.LineStart, LineEnd: inst.LineEnd,
		}

		if inst.ErrType == "missing" {
			out = append(out, OutError{
				ID: rid, GroupID: inst.GroupID, Code: inst.Code,
				SuggestedFix: inst.SuggestedFix, Rationale: inst.Rationale,
			})
			rep.Status = "skipped"
			rep.Reason = "missing: общий комментарий, без вставки в HTML"
			report = append(report, rep)
			continue
		}
		if inst.ErrType != "invalid" {
			rep.Status = "skipped"
			rep.Reason = "неподдерживаемый err_type"
			report = append(report, rep)
			continue
		}

		// кандидаты по линиям
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
		ids := make([]string, 0, len(candidates))
		for _, c := range candidates {
			ids = append(ids, c.Map.ElementID)
		}
		rep.CandidateIDs = ids

		// поиск лучшего совпадения, с учётом контейнеров
		type pick struct {
			b   *blockWork
			sub *blockWork // если спустились в leaf-фрагмент
			m   match
		}
		var best pick

		for _, b := range candidates {
			// определим тег верхнего уровня блока
			topTag := detectTopTagName(b.HTML) // реализуй через html.Parse: первый ElementNode name
			searchTargets := []*blockWork{b}

			if isContainer(topTag) {
				// разбиваем на листья и ищем в каждом
				leaves, err := collectLeafFragments(b.HTML)
				if err != nil {
					log.Warn("collectLeafFragments failed", "id", b.Map.ElementID, "err", err)
				} else {
					searchTargets = nil
					for _, leaf := range leaves {
						pi, err := buildPlainAndIndexFromHTML(leaf)
						if err != nil {
							continue
						}
						// создаём «виртуальный» под-блок
						sub := &blockWork{
							Map: markdown_service_client.Mapping{
								ElementID:     b.Map.ElementID, // тот же, но html будет другой
								HtmlContent:   leaf,
								MarkdownStart: b.Map.MarkdownStart,
								MarkdownEnd:   b.Map.MarkdownEnd,
							},
							Plain: pi,
							HTML:  leaf,
						}
						searchTargets = append(searchTargets, sub)
					}
				}
			}

			// ищем
			for _, target := range searchTargets {
				m := findBestMatch(inst.Snippet, target.Plain.Plain)
				if m.Found && (best.b == nil || m.Score > best.m.Score) {
					if target == b {
						best = pick{b: b, sub: nil, m: m}
					} else {
						best = pick{b: b, sub: target, m: m}
					}
				}
			}
		}

		// фиксация OutError (всегда добавляем)
		out = append(out, OutError{
			ID: rid, GroupID: inst.GroupID, Code: inst.Code,
			SuggestedFix: inst.SuggestedFix, Rationale: inst.Rationale,
		})

		if best.b == nil {
			rep.Status = "not-found"
			rep.Reason = "совпадение не найдено ни в одном кандидате"
			report = append(report, rep)
			continue
		}

		// целевой «блок для вставки»
		target := best.sub
		host := best.b
		if target == nil {
			target = best.b
		}

		// проверим, что матч попадает ВНУТРЬ одного текстового фрагмента (не «сшивает» несколько детей)
		// У нас target — это leaf outerHTML (li/p/span). В нём индекс/вставка безопасны.
		startPlain := clamp(best.m.Start, 0, len(target.Plain.IndexMap)-1)
		endPlain := clamp(best.m.End-1, 0, len(target.Plain.IndexMap)-1)
		startByte := target.Plain.IndexMap[startPlain]
		endByte := target.Plain.IndexMap[endPlain] + 1

		// применим вставку к target.HTML (локально)
		wrapped, werr := wrapHTMLSpan(target.HTML, startByte, endByte, rid)
		if werr != nil {
			rep.Status = "skipped"
			rep.Reason = "ошибка wrapHTMLSpan: " + werr.Error()
			rep.ElementID = host.Map.ElementID
			report = append(report, rep)
			continue
		}
		// теперь нужно заменить ЛИШЬ этот leaf обратно в host.HTML
		if best.sub != nil {
			// точечная подмена leaf в host.HTML
			host.HTML = strings.Replace(host.HTML, target.HTML, wrapped, 1)
		} else {
			host.HTML = wrapped
		}
		// и пересобрать host.Plain, чтобы последующие попадания были корректные (не обязательно, но полезно)
		if pi, err := buildPlainAndIndexFromHTML(host.HTML); err == nil {
			host.Plain = pi
		}

		rep.Status = "found"
		rep.ElementID = host.Map.ElementID
		report = append(report, rep)
		log.Info("invalid matched",
			"id", rid, "element", host.Map.ElementID,
			"score", best.m.Score, "byteStart", startByte, "byteEnd", endByte,
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
		return
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

	finalHTML, err = doc.Html()
	reportStr = buildReport(report)
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
	return htmlBlock[:startByte] + `<span error-id="` + spanID + `">` +
		htmlBlock[startByte:endByte] + `</span>` + htmlBlock[endByte:], nil
}

func detectTopTagName(htmlBlock string) string {
	n, err := html.Parse(strings.NewReader(htmlBlock))
	if err != nil {
		return ""
	}
	// найти первый ElementNode под корнем
	var tag string
	var dfs func(*html.Node)
	dfs = func(x *html.Node) {
		if tag != "" {
			return
		}
		if x.Type == html.ElementNode {
			tag = x.Data
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			dfs(c)
		}
	}
	dfs(n)
	return tag
}
