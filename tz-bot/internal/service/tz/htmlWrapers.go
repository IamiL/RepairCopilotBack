package tzservice

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

func WrapSubstring(s, sub, id string) (string, bool) {
	if sub == "" {
		return s, false
	}
	if !strings.Contains(s, sub) {
		return s, false
	}
	wrapped := `<span error-id="` + html.EscapeString(id) + `">` + sub + `</span>`
	return strings.ReplaceAll(s, sub, wrapped), true
}

// WrapApproxHTML ищет подстроку sub в текстовом содержимом htmlStr,
// позволяя подстроке "жить" частично внутри html-тегов (т.е. игнорируя теги при поиске).
// Если найдена, оборачивает соответствующий диапазон узлов/частей узлов в <span error-id="id">...</span>.
// Возвращает (результат, найдена_ли_подстрока, error).
func WrapSubstringApproxHTML(htmlStr, subSrt, id string) (string, bool, error) {
	if subSrt == "" {
		return htmlStr, false, errors.New("subSrt is empty")
	}

	// Дополнительная валидация: подстрока не должна быть слишком короткой (защита от случайных символов)
	if len(strings.TrimSpace(subSrt)) < 2 {
		return htmlStr, false, errors.New("subSrt too short after trimming")
	}

	pat, err := buildSpanningPattern(subSrt)
	if err != nil {
		return htmlStr, false, err
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return htmlStr, false, err
	}

	loc := re.FindStringIndex(htmlStr)
	if loc == nil {
		return htmlStr, false, nil
	}

	start, end := loc[0], loc[1]
	matchedContent := htmlStr[start:end]
	
	// Валидация: проверяем разумность найденного совпадения
	if !isReasonableMatch(matchedContent, subSrt) {
		return htmlStr, false, errors.New("найденное совпадение слишком длинное или содержит слишком много HTML")
	}

	wrapped := htmlStr[:start] +
		`<span error-id="` + html.EscapeString(id) + `">` +
		matchedContent +
		`</span>` +
		htmlStr[end:]

	return wrapped, true, nil
}

// buildSpanningPattern строит regex, который позволяет встречаться HTML-тегам
// перед первым символом и между КАЖДЫМИ двумя символами подстроки.
// Пробельные символы в подстроке сворачиваются в (?:\s|&nbsp;)+.
func buildSpanningPattern(sub string) (string, error) {
	var b strings.Builder
	// Разрешаем сразу перед первым символом ограниченное количество тегов (до 3)
	b.WriteString(`(?:<[^>]*>){0,3}`)

	rs := []rune(sub)
	for i, r := range rs {
		if unicode.IsSpace(r) {
			// Ограничиваем количество пробельных символов разумным пределом
			b.WriteString(`(?:\s|&nbsp;){1,5}`)
		} else {
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
		// Разрешаем ограниченное количество тегов между символами (до 2)
		if i != len(rs)-1 {
			b.WriteString(`(?:<[^>]*>){0,2}`)
		}
	}

	return b.String(), nil
}

// isReasonableMatch проверяет, является ли найденное совпадение разумным
func isReasonableMatch(matchedContent, originalSubstr string) bool {
	// 1. Проверяем соотношение длины
	originalLen := len(originalSubstr)
	matchedLen := len(matchedContent)
	
	// Совпадение не должно быть более чем в 5 раз длиннее оригинальной подстроки
	if matchedLen > originalLen * 5 {
		return false
	}
	
	// Совпадение не должно быть абсолютно слишком длинным (более 1000 символов)
	if matchedLen > 1000 {
		return false
	}

	// 2. Подсчитываем количество HTML-тегов в совпадении
	tagCount := strings.Count(matchedContent, "<")
	
	// Не должно быть слишком много тегов (более 10)
	if tagCount > 10 {
		return false
	}

	// 3. Проверяем, что совпадение не содержит слишком много блочных тегов
	blockTags := []string{"<p>", "<div>", "<section>", "<article>", "<header>", "<footer>"}
	blockTagCount := 0
	for _, tag := range blockTags {
		blockTagCount += strings.Count(strings.ToLower(matchedContent), tag)
	}
	
	// Не должно быть более 3 блочных тегов
	if blockTagCount > 3 {
		return false
	}

	// 4. Извлекаем текст без HTML и проверяем, что он похож на оригинал
	textContent := extractTextFromHTML(matchedContent)
	originalText := strings.TrimSpace(originalSubstr)
	
	// Текстовое содержимое не должно быть значительно длиннее оригинала
	if len(textContent) > len(originalText) * 3 {
		return false
	}

	// 5. Проверяем, что извлеченный текст содержит основную часть оригинальной подстроки
	normalizedOriginal := normalizeText(originalText)
	normalizedExtracted := normalizeText(textContent)
	
	if len(normalizedOriginal) > 0 && !strings.Contains(normalizedExtracted, normalizedOriginal[:len(normalizedOriginal)/2]) {
		return false
	}

	return true
}

// extractTextFromHTML грубо извлекает текст из HTML, удаляя теги
func extractTextFromHTML(htmlContent string) string {
	// Простое удаление тегов с помощью регулярного выражения
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(htmlContent, " ")
	
	// Заменяем &nbsp; на пробелы
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	// Схлопываем множественные пробелы
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	
	return strings.TrimSpace(text)
}

// normalizeText нормализует текст для сравнения
func normalizeText(text string) string {
	// Убираем лишние пробелы и переводим в нижний регистр
	re := regexp.MustCompile(`\s+`)
	normalized := re.ReplaceAllString(strings.TrimSpace(text), " ")
	return strings.ToLower(normalized)
}
