package tzservice

//
//import (
//	"fmt"
//	"regexp"
//	"strings"
//	"unicode"
//
//	"golang.org/x/net/html"
//)
//
//func clamp(v, lo, hi int) int {
//	if v < lo {
//		return lo
//	}
//	if v > hi {
//		return hi
//	}
//	return v
//}
//func mapValues[K comparable, V any](m map[K]V) []V {
//	a := make([]V, 0, len(m))
//	for _, v := range m {
//		a = append(a, v)
//	}
//	return a
//}
//
//// помощники: нормализация/токенизация
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
//
//func stripMarkdown(s string) string {
//	// очень лёгкая чистка markdown (для сниппетов):
//	// **bold**, *i*, __u__, `code`, [text](url), начальные "- " / "* " списков
//	s = regexp.MustCompile("`([^`]*)`").ReplaceAllString(s, "$1")
//	s = strings.NewReplacer("**", "", "__", "", "*", "", "_", "").Replace(s)
//	s = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(s, "$1")
//	s = regexp.MustCompile(`(?m)^\s*[-*]\s+`).ReplaceAllString(s, "")
//	return s
//}
//
//type plainIndex struct {
//	// mapping from plain-text index -> byte offset in original HTML
//	// we будем хранить только границы, нам хватит массива длиной len(plain)+1,
//	// где indexMap[i] = байтовая позиция в HTML до символа plain[i]
//	IndexMap []int
//	Plain    string
//}
//
//// вытащить текст из HTML и построить индексную карту
//func buildPlainAndIndexFromHTML(htmlBlock string) (plainIndex, error) {
//	// Парсим HTML как фрагмент
//	node, err := html.Parse(strings.NewReader(htmlBlock))
//	if err != nil {
//		return plainIndex{}, err
//	}
//
//	var buf strings.Builder
//	var indexMap []int
//	var walk func(*html.Node, int) int
//	walk = func(n *html.Node, htmlPos int) int {
//		// htmlPos — текущее смещение по исходной HTML-строке (байты).
//		// Для простоты считаем, что htmlBlock — исходная строка;
//		// будем инкрементировать htmlPos, когда пишем в buf.
//		// Реализуем через повторный рендер ноды до текста? Проще: пробежимся текст-ноды.
//		if n.Type == html.TextNode {
//			text := n.Data
//			// перед добавлением: нормализация пробелов позже; сейчас точный маппинг
//			for i := 0; i < len(text); i++ {
//				// добавляем символ в буфер
//				buf.WriteByte(text[i])
//				indexMap = append(indexMap, htmlPos+i) // позиция в html перед символом
//			}
//			return htmlPos + len(text)
//		}
//		for c := n.FirstChild; c != nil; c = c.NextSibling {
//			htmlPos = walk(c, htmlPos)
//		}
//		return htmlPos
//	}
//	// Примерная оценка смещения: для простоты считаем нулём,
//	// т.к. мы будем вставлять по indexMap в конкретном htmlBlock,
//	// без привязки к глобальному документу.
//	walk(node, 0)
//
//	plain := buf.String()
//	// теперь нормализуем plain так же, как нормализуем сниппеты
//	// но, чтобы не потерять карту, пересоберём карту под нормализованный текст
//	normalized := normalizeText(plain)
//	// Перестроим карту "как сможем": делаем двухуказательный проход по plain и normalized
//	// Схлопывание пробелов может склеить позиции — возьмём первую попавшуюся
//	newIndex := make([]int, 0, len(normalized))
//	i, j := 0, 0
//	for i < len(plain) && j < len(normalized) {
//		pc := plain[i]
//		nc := normalized[j]
//		// приводим регистр?
//		if pc == nc {
//			newIndex = append(newIndex, indexMap[i])
//			i++
//			j++
//			continue
//		}
//		// если plain[i] пробел и в normalized уже схлопнуто — пропускаем лишние пробелы
//		if unicode.IsSpace(rune(pc)) {
//			i++
//			continue
//		}
//		if unicode.IsSpace(rune(nc)) {
//			// normalized вставил пробел — попробуем пропустить шум в plain до ближайшего непробельного
//			i++
//			continue
//		}
//		// иное несовпадение: сдвигаемся в plain (теги/шумы уже выкинуты)
//		i++
//	}
//	// защитимся от пустоты
//	if len(newIndex) == 0 && len(normalized) == 0 {
//		newIndex = []int{0}
//	}
//	return plainIndex{IndexMap: newIndex, Plain: normalized}, nil
//}
//
//// токенизация сниппета
//func tokens(s string) []string {
//	s = stripMarkdown(s)
//	s = normalizeText(s)
//	re := regexp.MustCompile(`[A-Za-zА-Яа-я0-9]+|[.,:;!?()-]`)
//	toks := re.FindAllString(s, -1)
//	// выкинем очень короткие/малозначимые токены (кроме пунктуации из правил)
//	out := make([]string, 0, len(toks))
//	for _, t := range toks {
//		if len([]rune(t)) >= 2 || regexp.MustCompile(`[.,:;!?()-]`).MatchString(t) {
//			out = append(out, t)
//		}
//	}
//	return out
//}
//
//// последовательный поиск токенов с допуском пробелов/шума
//type match struct {
//	Start, End int
//	Score      int
//	Found      bool
//}
//
//func findBestMatch(snippet string, plain string) match {
//	toks := tokens(snippet)
//	if len(toks) == 0 || len(plain) == 0 {
//		return match{}
//	}
//	// простая эвристика: строим регекс вида \btok1\b.*?\btok2\b.*?\btok3\b ...
//	// (.*? допускает пробелы/шум); но plain уже без тегов, с норм пробелами.
//	// Добавим \s+ вместо .*, чтобы не “переедать”
//	var b strings.Builder
//	b.WriteString(`(?s)`)
//	for i, t := range toks {
//		if i > 0 {
//			b.WriteString(`\s+`)
//		}
//		b.WriteString(`\Q` + t + `\E`)
//	}
//	re := regexp.MustCompile(b.String())
//	loc := re.FindStringIndex(plain)
//	if loc != nil {
//		return match{Start: loc[0], End: loc[1], Score: loc[1] - loc[0], Found: true}
//	}
//	// fallback: попробуем обрезать крайние токены
//	for cut := 1; cut <= 2 && len(toks)-cut >= 2; cut++ {
//		sub := strings.Join(toks[0:len(toks)-cut], " ")
//		re2 := regexp.MustCompile(`(?s)\Q` + sub + `\E`)
//		loc2 := re2.FindStringIndex(plain)
//		if loc2 != nil {
//			return match{Start: loc2[0], End: loc2[1], Score: loc2[1] - loc2[0], Found: true}
//		}
//	}
//	return match{}
//}
//
//// оборачивание диапазона в HTML (по байтовым индексам)
//func wrapHTMLSpan(htmlBlock string, startByte, endByte int, spanID string) (string, error) {
//	if startByte < 0 || endByte > len(htmlBlock) || startByte >= endByte {
//		return htmlBlock, fmt.Errorf("bad range")
//	}
//	return htmlBlock[:startByte] + `<span error-id="` + spanID + `">` +
//		htmlBlock[startByte:endByte] + `</span>` + htmlBlock[endByte:], nil
//}
//func detectTopTagName(htmlBlock string) string {
//n, err := html.Parse(strings.NewReader(htmlBlock))
//if err != nil { return "" }
//// найти первый ElementNode под корнем
//var tag string
//var dfs func(*html.Node)
//dfs = func(x *html.Node) {
//if tag != "" { return }
//if x.Type == html.ElementNode {
//tag = x.Data
//return
//}
//for c := x.FirstChild; c != nil; c = c.NextSibling { dfs(c) }
//}
//dfs(n)
//return tag
//}
