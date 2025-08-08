package tzservice

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

//func informativeTokens(snippet string, maxTokens int) []string {
//	toks := tokens(snippet) // уже stripMarkdown + normalizeText внутри tokens
//	sort.SliceStable(toks, func(i, j int) bool { return len(toks[i]) > len(toks[j]) })
//	// убираем чистую пунктуацию
//	out := make([]string, 0, len(toks))
//	for _, t := range toks {
//		if regexp.MustCompile(`^[\p{L}\p{N}]{2,}$`).MatchString(t) {
//			out = append(out, t)
//		}
//		if len(out) == maxTokens {
//			break
//		}
//	}
//	if len(out) == 0 {
//		return tokens(snippet)
//	}
//	// сохраняем исходный порядок inform-токенов, как в сниппете
//	order := map[string]int{}
//	for i, tok := range tokens(snippet) {
//		if _, ok := order[tok]; !ok {
//			order[tok] = i
//		}
//	}
//	sort.SliceStable(out, func(i, j int) bool { return order[out[i]] < order[out[j]] })
//	return out
//}

//func regexForTokens(toks []string) *regexp.Regexp {
//	// \b tok1 \b (?:\W+|\s+)* \b tok2 \b ...
//	// где \W допускает пунктуацию; уже в normalizeText всё в нижнем регистре
//	var b strings.Builder
//	b.WriteString(`(?s)`)
//	for i, t := range toks {
//		if i > 0 {
//			b.WriteString(`(?:\W+|\s+)*`)
//		}
//		b.WriteString(`\b` + regexp.QuoteMeta(t) + `\b`)
//	}
//	return regexp.MustCompile(b.String())
//}

type match struct {
	Start, End int
	Found      bool
}

func findBestMatch(snippet, plainNorm string) match {
	if plainNorm == "" {
		return match{}
	}
	toksCore := informativeTokens(snippet, 6)
	if len(toksCore) >= 2 {
		re := regexForTokens(toksCore)
		if loc := re.FindStringIndex(plainNorm); loc != nil {
			return match{Start: loc[0], End: loc[1], Found: true}
		}
	}
	// полный набор
	toksAll := tokensNormalized(snippet)
	if len(toksAll) >= 1 {
		re2 := regexForTokens(toksAll)
		if loc := re2.FindStringIndex(plainNorm); loc != nil {
			return match{Start: loc[0], End: loc[1], Found: true}
		}
	}
	// укороченные с "…"
	raw := strings.TrimSpace(stripMarkdown(snippet))
	if strings.HasSuffix(raw, "...") || strings.HasSuffix(raw, "…") {
		t3 := informativeTokens(snippet, 3)
		if len(t3) >= 1 {
			re3 := regexForTokens(t3)
			if loc := re3.FindStringIndex(plainNorm); loc != nil {
				return match{Start: loc[0], End: loc[1], Found: true}
			}
		}
	}
	// скользящее окно
	base := tokensNormalized(snippet)
	win := 4
	if len(base) < win {
		win = len(base)
	}
	for i := 0; i+win <= len(base); i++ {
		re := regexForTokens(base[i : i+win])
		if loc := re.FindStringIndex(plainNorm); loc != nil {
			return match{Start: loc[0], End: loc[1], Found: true}
		}
	}
	return match{}
}

// add near normalizeText
//var confusables = map[rune]rune{
//	// латиница <-> кириллица, самые болезненные
//	'А': 'A', 'В': 'B', 'Е': 'E', 'К': 'K', 'М': 'M', 'Н': 'H', 'О': 'O', 'Р': 'P', 'С': 'C', 'Т': 'T', 'Х': 'X', 'У': 'Y',
//	'a': 'a', 'е': 'e', 'к': 'k', 'м': 'm', 'н': 'h', 'o': 'o', 'р': 'p', 'с': 'c', 'т': 't', 'х': 'x', 'у': 'y',
//	// обратные (лат->кир) не нужны — приводим всё к латинице
//}
//
//func normalizeSymbols(s string) string {
//	// типографика и единицы
//	repl := map[string]string{
//		// кавычки/многоточие/тире
//		"“": "\"", "”": "\"", "„": "\"", "«": "\"", "»": "\"", "‘": "'", "’": "'", "…": "...",
//		"—": "-", "–": "-",
//		// пробелы
//		"\u00A0": " ", "\u2009": " ", "\u202F": " ",
//		// матем/единицы
//		"×": "x", "·": "x", "˚": "°",
//		"″": "\"", "’’": "\"", "”": "\"", // иногда дюймы/секунды угла
//	}
//	for k, v := range repl {
//		s = strings.ReplaceAll(s, k, v)
//	}
//
//	// нормализация °C / ° C / градусы
//	s = regexp.MustCompile(`\s*°\s*[CС]`).ReplaceAllString(s, "°C")         // C и кир. С
//	s = regexp.MustCompile(`\b([0-9]+)\s*["”″]`).ReplaceAllString(s, `$1"`) // 55” -> 55"
//	s = regexp.MustCompile(`\b4\s*[KК]\b`).ReplaceAllString(s, "4K")        // 4К -> 4K
//
//	// Приведём похожие буквы к латинице (важно для 1С/1C, S/С и т.д.)
//	var b strings.Builder
//	for _, r := range s {
//		if rr, ok := confusables[r]; ok {
//			b.WriteRune(rr)
//		} else {
//			b.WriteRune(r)
//		}
//	}
//	return b.String()
//}
//
//func normalizeText(s string) string {
//	s = html.UnescapeString(s)
//	s = normalizeSymbols(s)
//	s = strings.ToLower(s) // регистр не важен
//	s = strings.TrimSpace(s)
//	reSpaces := regexp.MustCompile(`\s+`)
//	s = reSpaces.ReplaceAllString(s, " ")
//	return s
//}
//
//func stripMarkdown(s string) string {
//	s = regexp.MustCompile("`([^`]*)`").ReplaceAllString(s, "$1")
//	s = strings.NewReplacer("**", "", "__", "", "*", "", "_", "").Replace(s)
//	s = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(s, "$1")
//	s = regexp.MustCompile(`(?m)^\s*[-*]\s+`).ReplaceAllString(s, "") // - bullets
//	return s
//}

// Вернёт диапазон в PlainOrig, который соответствует [startNorm,endNorm) в PlainNorm
func mapNormToOrig(plainOrig, plainNorm string, startNorm, endNorm int) (int, int) {
	// два указателя, как «merge»:
	iOrig, iNorm := 0, 0
	startOrig, endOrig := -1, -1

	for iOrig < len(plainOrig) && iNorm < len(plainNorm) {
		cO := runeAt(plainOrig, &iOrig)
		cN := runeAt(plainNorm, &iNorm)

		// сравниваем после normalizeText(cO) vs cN — но normalizeText меняет строку целиком.
		// Проще: нормализуем посимвольно: приводим cO к "скалярной" форме для сравнения:
		nO := normalizeRune(cO)

		if nO == cN {
			if iNorm-1 == startNorm && startOrig == -1 {
				startOrig = iOrig - utf8.RuneLen(cO)
			}
			if iNorm == endNorm {
				endOrig = iOrig - utf8.RuneLen(cO)
				break
			}
			continue
		}

		// если расхождение — продвигаем Orig, пока не синхронизируемся
		// (может быть пунктуация/пробелы в одной строке и другой)
		// Упростим: если cN — пробельный/неалфанум — разрешим пропуски
		if unicode.IsSpace(cN) || !unicode.IsLetter(cN) && !unicode.IsNumber(cN) {
			// сдвигаем норм до ближайшего значимого
			for iNorm < len(plainNorm) {
				r := runeAt(plainNorm, &iNorm)
				if unicode.IsLetter(r) || unicode.IsNumber(r) {
					iNorm -= utf8.RuneLen(r)
					break
				}
			}
			continue
		}
		// иначе двигаем Orig
	}
	if startOrig == -1 {
		startOrig = 0
	}
	if endOrig == -1 {
		endOrig = len(plainOrig)
	}
	return startOrig, endOrig
}

// черновые хелперы
func runeAt(s string, i *int) rune {
	r, size := utf8.DecodeRuneInString(s[*i:])
	*i += size
	return r
}

func normalizeRune(r rune) rune {
	// упрощённо: приводим к нижнему регистру и нормализуем confusables из предыдущей версии
	r = unicode.ToLower(r)
	if rr, ok := confusables[r]; ok {
		return rr
	}
	switch r {
	case '“', '”', '„', '«', '»':
		return '"'
	case '‘', '’':
		return '\''
	case '—', '–':
		return '-'
	}
	return r
}

func wrapInLeafHTML(leafHTML string, plainOrig string, startOrig, endOrig int, id string) (string, error) {
	// найдём в leafHTML подстроку plainOrig[startOrig:endOrig] через «текстовый поиск»,
	// предварительно выкинув теги? Проще: найдём N-е вхождение последовательности с учётом того,
	// что leafHTML содержит тот же порядок символов текста. Начнём с прямого поиска:
	sub := plainOrig[startOrig:endOrig]
	if sub == "" {
		return leafHTML, fmt.Errorf("empty sub")
	}
	idx := strings.Index(leafHTML, sub)
	if idx < 0 {
		// fallback: схлопнём пробелы у обоих и попробуем по схлопнутым, а потом вернуть к исходным индексам…
		// Чтобы не усложнять — вернём ошибку; в отчёт попадёт reason.
		return leafHTML, fmt.Errorf("sub not found in leaf html")
	}
	return leafHTML[:idx] + `<span data-error="` + id + `">` + sub + `</span>` + leafHTML[idx+len(sub):], nil
}

func detectTopTagName(htmlBlock string) string {
	n, err := html.Parse(strings.NewReader(htmlBlock))
	if err != nil {
		return ""
	}
	body := findBody(n)
	if body == nil {
		body = n
	}
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			return strings.ToLower(c.Data)
		}
	}
	return ""
}
