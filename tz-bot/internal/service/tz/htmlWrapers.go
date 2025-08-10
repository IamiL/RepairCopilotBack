package tzservice

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

func WrapSubstringSimilar(s, sub, id string) (string, bool) {
	if sub == "" {
		return s, false
	}
	if !strings.Contains(s, sub) {
		return s, false
	}
	wrapped := `<span error-id="` + html.EscapeString(id) + `">` + sub + `</span>`
	return strings.ReplaceAll(s, sub, wrapped), true
}

/*
Корневая функция:
  - Пытается найти точное вхождение во всём документе (дешёвый путь).
  - Если нет, ищет подстроку в минимальных блочных контейнерах (<p>, <li>, <td>...).
  - Внутри контейнера сначала пробует точное вхождение,
    затем приближённый поиск, разрешая пересекать только инлайновые теги.
  - Глобального «приближённого» поиска по всему документу НЕТ, чтобы не
    оборачивать большие фрагменты и не пересекать блоки.

Возвращает: (изменённыйHTML, нашлось, error)
*/
func WrapSubstringSmartHTML(htmlStr, subStr, id string) (string, bool, error) {
	if subStr == "" {
		return htmlStr, false, errors.New("subStr is empty")
	}
	if len(strings.TrimSpace(subStr)) < 2 {
		return htmlStr, false, errors.New("subStr too short after trimming")
	}

	// 1) Самый простой случай — точное вхождение в документе
	if strings.Contains(htmlStr, subStr) {
		return WrapSubstring(htmlStr, subStr, id), true, nil
	}

	// 2) Ищем внутри минимальных блочных контейнеров
	wrapped, found, err := findAndWrapMinimalBlock(htmlStr, subStr, id)
	if err != nil {
		return htmlStr, false, err
	}
	return wrapped, found, nil
}

/* -------------------------- Вспомогательные части -------------------------- */

// Точное оборачивание первого вхождения (без регулярок).
func WrapSubstring(htmlStr, subStr, id string) string {
	i := strings.Index(htmlStr, subStr)
	if i < 0 {
		return htmlStr
	}
	return htmlStr[:i] +
		`<span error-id="` + html.EscapeString(id) + `">` +
		subStr +
		`</span>` +
		htmlStr[i+len(subStr):]
}

// Поиск «минимального» блока среди распространённых блочных тегов.
// Здесь нет глобального fallback к приблизённому поиску.
func findAndWrapMinimalBlock(htmlStr, subStr, id string) (string, bool, error) {
	blockTags := []string{
		"p", "div", "section", "article",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"li", "td", "th",
	}

	for _, tag := range blockTags {
		if wrapped, found := findInBlocks(htmlStr, subStr, id, tag); found {
			return wrapped, true, nil
		}
	}
	return htmlStr, false, nil
}

// Ищет по блокам заданного типа и, если в блоке действительно есть
// искомый текст, оборачивает внутри него (точно или «инлайново-приближённо»).
func findInBlocks(htmlStr, subStr, id, tagName string) (string, bool) {
	// (?i) — регистронезависимо, (?s) — '.' матчит переводы строк
	tagPattern := fmt.Sprintf(`(?is)(<(%s)\b[^>]*>)(.*?)(</\2>)`, regexp.QuoteMeta(tagName))
	re, err := regexp.Compile(tagPattern)
	if err != nil {
		return htmlStr, false
	}

	matches := re.FindAllStringSubmatchIndex(htmlStr, -1)
	if matches == nil {
		return htmlStr, false
	}

	// Работаем слева направо: как только удачно обернули — выходим.
	for _, loc := range matches {
		// индексы групп: 0:full, 1-2:full idxs, 3-4:openTag, 5-6:tagName, 7-8:content, 9-10:closeTag
		if len(loc) < 10 {
			continue
		}

		openStart, openEnd := loc[3], loc[4]
		contentStart, contentEnd := loc[7], loc[8]
		closeStart, closeEnd := loc[9], loc[10]

		fullBlock := htmlStr[loc[0]:loc[1]]
		openTag := htmlStr[openStart:openEnd]
		blockContent := htmlStr[contentStart:contentEnd]
		closeTag := htmlStr[closeStart:closeEnd]

		// Быстрая проверка по «текстовой» нормализации: есть ли искомое
		// (чтобы зря не запускать «инлайновую» регулярку)
		textContent := extractTextFromHTML(blockContent)
		if !strings.Contains(normalizeText(textContent), normalizeText(subStr)) {
			continue
		}
		if !isBlockSizeReasonable(blockContent, subStr) {
			continue
		}

		// Пытаемся обернуть внутри блока
		wrappedContent, found, err := wrapWithinBlock(blockContent, subStr, id)
		if err != nil || !found {
			continue
		}

		newBlock := openTag + wrappedContent + closeTag
		// Заменяем РОВНО один раз найденный fullBlock,
		// чтобы не затронуть похожие блоки дальше.
		result := strings.Replace(htmlStr, fullBlock, newBlock, 1)
		return result, true
	}
	return htmlStr, false
}

// Внутри блока: сначала точное, затем «инлайново-приближённое».
func wrapWithinBlock(blockContent, subStr, id string) (string, bool, error) {
	// 1) Точное вхождение
	if strings.Contains(blockContent, subStr) {
		wrapped := strings.Replace(blockContent, subStr,
			`<span error-id="`+html.EscapeString(id)+`">`+subStr+`</span>`, 1)
		return wrapped, true, nil
	}

	// 2) «Приближённый» поиск, разрешая пересекать ТОЛЬКО инлайновые теги.
	wrapped, found, err := wrapSubstringApproxInlineOnly(blockContent, subStr, id)
	if err != nil {
		return blockContent, false, err
	}
	return wrapped, found, nil
}

/* ---------------- «Инлайново-приближённый» поиск ---------------- */

var inlineTags = []string{
	"a", "abbr", "b", "bdi", "bdo", "br", "cite", "code", "data", "dfn", "em", "i", "kbd",
	"mark", "q", "rp", "rt", "rtc", "ruby", "s", "samp", "small", "span", "strong",
	"sub", "sup", "time", "u", "var", "wbr", "del", "ins",
}

func inlineAlternation() string {
	return strings.Join(inlineTags, "|")
}

func buildInlineSpanningPattern(sub string) string {
	// (?i) регистронезависимо, (?s) '.' матчит '\n'
	var b strings.Builder
	b.WriteString(`(?is)`)

	inlineTag := `(?:<(?:` + inlineAlternation() + `)(?:\s[^>]*?)?>)`

	// Небольшой зазор инлайновых тегов перед первым символом
	b.WriteString(`(?:` + inlineTag + `){0,3}`)

	rs := []rune(sub)
	for i, r := range rs {
		if unicode.IsSpace(r) {
			b.WriteString(`(?:\s|&nbsp;){1,5}`)
		} else {
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
		if i != len(rs)-1 {
			// Между символами допустимы только инлайновые теги (немного)
			b.WriteString(`(?:` + inlineTag + `){0,5}`)
		}
	}
	return b.String()
}

var blockTagRe = regexp.MustCompile(`(?i)</?(?:p|div|section|article|header|footer|ul|ol|li|table|thead|tbody|tr|td|th|h[1-6])\b`)

func wrapSubstringApproxInlineOnly(htmlStr, subStr, id string) (string, bool, error) {
	if subStr == "" {
		return htmlStr, false, errors.New("subStr is empty")
	}
	if len(strings.TrimSpace(subStr)) < 2 {
		return htmlStr, false, errors.New("subStr too short after trimming")
	}

	pat := buildInlineSpanningPattern(subStr)
	re, err := regexp.Compile(pat)
	if err != nil {
		return htmlStr, false, err
	}

	loc := re.FindStringIndex(htmlStr)
	if loc == nil {
		return htmlStr, false, nil
	}

	start, end := loc[0], loc[1]
	matched := htmlStr[start:end]

	// Защита: совпадение не должно пересекать блочные теги
	if blockTagRe.MatchString(matched) {
		return htmlStr, false, nil
	}

	if !isReasonableMatch(matched, subStr) {
		return htmlStr, false, errors.New("approx match looks unreasonable")
	}

	wrapped := htmlStr[:start] +
		`<span error-id="` + html.EscapeString(id) + `">` +
		matched +
		`</span>` +
		htmlStr[end:]
	return wrapped, true, nil
}

/* ------------------------- Валидация совпадения ------------------------- */

// Грубая «разумность» для найденного фрагмента
func isReasonableMatch(matchedContent, originalSubstr string) bool {
	origLen := len(originalSubstr)
	matchLen := len(matchedContent)

	// Не длиннее чем в 5 раз (и не абсолютно слишком длинно)
	if matchLen > origLen*5 || matchLen > 1000 {
		return false
	}

	// Не слишком много тегов
	if strings.Count(matchedContent, "<") > 10 {
		return false
	}

	// Не слишком много блочных тегов
	if blockTagRe.MatchString(matchedContent) {
		return false
	}

	// Сравниваем «чистый» текст
	textContent := extractTextFromHTML(matchedContent)
	originalText := strings.TrimSpace(originalSubstr)
	if len(textContent) > len(originalText)*3 {
		return false
	}

	// Проверяем, что хотя бы половина нормализованного оригинала встречается
	normalizedOriginal := normalizeText(originalText)
	normalizedExtracted := normalizeText(textContent)
	if len(normalizedOriginal) > 0 {
		half := len(normalizedOriginal) / 2
		if half == 0 {
			half = 1
		}
		if !strings.Contains(normalizedExtracted, normalizedOriginal[:half]) {
			return false
		}
	}
	return true
}

// Разумность размера контейнера (чтобы не оборачивать «простыню»)
func isBlockSizeReasonable(blockContent, subStr string) bool {
	text := extractTextFromHTML(blockContent)
	if len(text) > len(subStr)*10 {
		return false
	}

	// Немного вложенных блочных тегов внутри — ок, но не слишком много
	nestedBlockCount := 0
	for _, tag := range []string{"<p", "<div", "<section", "<article", "<h1", "<h2", "<h3", "<h4", "<h5", "<h6"} {
		nestedBlockCount += strings.Count(strings.ToLower(blockContent), strings.ToLower(tag))
	}
	return nestedBlockCount <= 2
}

/* ------------------ Извлечение и нормализация текста ------------------ */

// Очень простой «стриппер» тегов для эвристик.
func extractTextFromHTML(s string) string {
	reTags := regexp.MustCompile(`<[^>]*>`)
	txt := reTags.ReplaceAllString(s, " ")
	txt = strings.ReplaceAll(txt, "&nbsp;", " ")
	reWS := regexp.MustCompile(`\s+`)
	txt = reWS.ReplaceAllString(txt, " ")
	return strings.TrimSpace(txt)
}

func normalizeText(s string) string {
	reWS := regexp.MustCompile(`\s+`)
	return strings.ToLower(reWS.ReplaceAllString(strings.TrimSpace(s), " "))
}
