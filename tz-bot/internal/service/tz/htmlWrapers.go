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
	tagPattern := fmt.Sprintf(`(?is)(<(%s)\b[^>]*>)(.*?)(</%s>)`, regexp.QuoteMeta(tagName), regexp.QuoteMeta(tagName))
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
		// индексы групп: 0-1:full, 2-3:openTag, 4-5:tagName, 6-7:content, 8-9:closeTag  
		if len(loc) < 10 {
			continue
		}

		openStart, openEnd := loc[2], loc[3]
		contentStart, contentEnd := loc[6], loc[7]
		closeStart, closeEnd := loc[8], loc[9]

		fullBlock := htmlStr[loc[0]:loc[1]]
		openTag := htmlStr[openStart:openEnd]
		blockContent := htmlStr[contentStart:contentEnd]
		closeTag := htmlStr[closeStart:closeEnd]

		// Быстрая проверка по «текстовой» нормализации: есть ли искомое
		// (чтобы зря не запускать «инлайновую» регулярку)
		textContent := extractTextFromHTML(blockContent)
		normalizedText := normalizeText(textContent)
		normalizedSubStr := normalizeText(subStr)
		
		// Проверяем точное содержание или частичное совпадение
		hasExactMatch := strings.Contains(normalizedText, normalizedSubStr)
		hasPartialMatch := false
		
		if !hasExactMatch {
			// Проверяем частичное совпадение (большинство слов)
			subWords := strings.Fields(normalizedSubStr)
			matchingWords := 0
			for _, word := range subWords {
				if len(word) > 2 && strings.Contains(normalizedText, word) {
					matchingWords++
				}
			}
			hasPartialMatch = len(subWords) > 0 && float64(matchingWords)/float64(len(subWords)) > 0.7
		}
		
		if !hasExactMatch && !hasPartialMatch {
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

	// 2) Попытка обработки многоуровневых span-структур
	wrapped, found, err := wrapInNestedSpans(blockContent, subStr, id)
	if err == nil && found {
		return wrapped, true, nil
	}

	// 3) «Приближённый» поиск, разрешая пересекать ТОЛЬКО инлайновые теги.
	wrapped, found, err = wrapSubstringApproxInlineOnly(blockContent, subStr, id)
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

	// Не длиннее чем в 10 раз (увеличено для span-структур)
	if matchLen > origLen*10 || matchLen > 2000 {
		return false
	}

	// Подсчитываем только span теги для более точной оценки
	spanCount := strings.Count(matchedContent, "<span")
	totalTagCount := strings.Count(matchedContent, "<")
	
	// Разрешаем больше тегов, если это в основном span-ы
	maxTags := 30
	if spanCount > totalTagCount/2 {
		maxTags = 100 // для случаев с множественными span
	}
	
	if totalTagCount > maxTags {
		return false
	}

	// Не слишком много блочных тегов
	if blockTagRe.MatchString(matchedContent) {
		return false
	}

	// Сравниваем «чистый» текст
	textContent := extractTextFromHTML(matchedContent)
	originalText := strings.TrimSpace(originalSubstr)
	if len(textContent) > len(originalText)*5 {
		return false
	}

	// Проверяем схожесть текста
	normalizedOriginal := normalizeText(originalText)
	normalizedExtracted := normalizeText(textContent)
	
	if len(normalizedOriginal) > 0 {
		// Используем более гибкую проверку схожести
		similarity := calculateTextSimilarity(normalizedExtracted, normalizedOriginal)
		contains := strings.Contains(normalizedExtracted, normalizedOriginal)
		containsPartial := len(normalizedOriginal) >= 4 && 
			strings.Contains(normalizedExtracted, normalizedOriginal[:len(normalizedOriginal)/2])
		
		return similarity > 0.5 || contains || containsPartial
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
	// Сначала приводим к нижнему регистру
	s = strings.ToLower(strings.TrimSpace(s))
	
	// Убираем лишние пробелы
	reWS := regexp.MustCompile(`\s+`)
	s = reWS.ReplaceAllString(s, " ")
	
	// Дополнительная обработка для случаев типа "MES -система" -> "mes-система"
	// Убираем пробелы вокруг дефисов
	reDashes := regexp.MustCompile(`\s*-\s*`)
	s = reDashes.ReplaceAllString(s, "-")
	
	// Убираем пробелы после точек (для случаев типа ". MES" -> ".mes")
	reDots := regexp.MustCompile(`\.\s+`)
	s = reDots.ReplaceAllString(s, ".")
	
	// Убираем пробелы перед точками (для случаев типа "3 . 2" -> "3.2")
	reDotsAfter := regexp.MustCompile(`\s+\.`)
	s = reDotsAfter.ReplaceAllString(s, ".")
	
	return strings.TrimSpace(s)
}

/* ---------------- Обработка многоуровневых span-структур ---------------- */

// Структура для хранения информации о текстовых фрагментах в HTML
type textFragment struct {
	text       string
	htmlPos    int
	htmlLen    int
	normalized string
}

// Специализированная функция для поиска текста, разбитого на множественные span
func wrapInNestedSpans(htmlStr, subStr, id string) (string, bool, error) {
	if subStr == "" {
		return htmlStr, false, errors.New("subStr is empty")
	}

	// Нормализуем искомую строку для сравнения
	normalizedTarget := normalizeText(subStr)
	if len(normalizedTarget) < 2 {
		return htmlStr, false, errors.New("normalized subStr too short")
	}

	// Пытаемся найти начальную позицию текста методом скользящего окна
	position, length, err := findTextInSpanSequence(htmlStr, normalizedTarget)
	if err != nil || position == -1 {
		return htmlStr, false, err
	}

	// Проверяем, что найденный фрагмент разумен
	candidateHTML := htmlStr[position : position+length]
	if !isNestedSpanMatch(candidateHTML, subStr) {
		return htmlStr, false, nil
	}

	// Оборачиваем найденный фрагмент
	wrapped := htmlStr[:position] +
		`<span error-id="` + html.EscapeString(id) + `">` +
		candidateHTML +
		`</span>` +
		htmlStr[position+length:]

	return wrapped, true, nil
}

// Поиск текстовой последовательности в HTML, разбитой на span-ы
func findTextInSpanSequence(htmlStr, normalizedTarget string) (int, int, error) {
	targetWords := strings.Fields(normalizedTarget)
	if len(targetWords) == 0 {
		return -1, 0, errors.New("no target words")
	}

	// Более точное регулярное выражение для поиска span-тегов
	spanPattern := regexp.MustCompile(`(?i)<span[^>]*>([^<]*)</span>`)
	matches := spanPattern.FindAllStringSubmatchIndex(htmlStr, -1)
	
	if len(matches) == 0 {
		return -1, 0, nil
	}

	// Собираем ВСЕ span-ы (включая пробелы и пустые)
	var fragments []textFragment
	
	for _, match := range matches {
		if len(match) >= 4 {
			// match содержит: 0-1:полное совпадение, 2-3:содержимое span
			spanStart, spanEnd := match[0], match[1]
			contentStart, contentEnd := match[2], match[3]
			
			if contentStart != -1 && contentEnd != -1 {
				textContent := htmlStr[contentStart:contentEnd] // НЕ trimSpace!
				fragments = append(fragments, textFragment{
					text:       textContent,
					htmlPos:    spanStart,
					htmlLen:    spanEnd - spanStart,
					normalized: textContent, // Нормализацию делаем позже
				})
			}
		}
	}

	// Ищем последовательность фрагментов, которая содержит целевой текст
	return findBestSpanSequenceImproved(fragments, normalizedTarget)
}

// Поиск наилучшей последовательности span-ов
func findBestSpanSequence(fragments []textFragment, normalizedTarget string) (int, int, error) {
	if len(fragments) == 0 {
		return -1, 0, nil
	}

	targetWords := strings.Fields(normalizedTarget)
	if len(targetWords) == 0 {
		return -1, 0, nil
	}

	// Поиск лучшего совпадения с большим окном поиска
	bestStartIdx := -1
	bestEndIdx := -1
	bestSimilarity := 0.0

	// Скользящее окно по фрагментам
	for startIdx := 0; startIdx < len(fragments); startIdx++ {
		var combinedNormalized strings.Builder
		
		for endIdx := startIdx; endIdx < len(fragments) && endIdx < startIdx+100; endIdx++ {
			if combinedNormalized.Len() > 0 {
				combinedNormalized.WriteString(" ")
			}
			combinedNormalized.WriteString(fragments[endIdx].normalized)
			
			currentNormalized := combinedNormalized.String()
			
			// Проверяем различные типы совпадений
			exactMatch := strings.Contains(currentNormalized, normalizedTarget)
			similarity := calculateTextSimilarity(currentNormalized, normalizedTarget)
			
			// Проверяем частичные совпадения (если содержит большую часть слов)
			matchingWords := 0
			for _, word := range targetWords {
				if len(word) > 2 && strings.Contains(currentNormalized, word) {
					matchingWords++
				}
			}
			partialMatch := float64(matchingWords) / float64(len(targetWords))
			
			// Оценка качества совпадения
			score := similarity
			if exactMatch {
				score = 1.0 // Точное совпадение всегда лучше
			} else if partialMatch > 0.7 {
				score = partialMatch * 0.9 // Частичное совпадение тоже неплохо
			}
			
			if score > bestSimilarity && score > 0.6 {
				// Проверяем разумность длины
				startPos := fragments[startIdx].htmlPos
				endPos := fragments[endIdx].htmlPos + fragments[endIdx].htmlLen
				
				if endPos > startPos && endPos-startPos < len(normalizedTarget)*20 {
					bestSimilarity = score
					bestStartIdx = startIdx
					bestEndIdx = endIdx
					
					// Если нашли точное совпадение, можно остановиться
					if exactMatch {
						break
					}
				}
			}
			
			// Прекращаем, если текст стал слишком длинным
			if len(currentNormalized) > len(normalizedTarget)*5 {
				break
			}
		}
		
		// Если нашли точное совпадение, можно остановить поиск
		if bestSimilarity >= 1.0 {
			break
		}
	}

	if bestStartIdx == -1 || bestEndIdx == -1 {
		return -1, 0, nil
	}

	startPos := fragments[bestStartIdx].htmlPos
	endPos := fragments[bestEndIdx].htmlPos + fragments[bestEndIdx].htmlLen
	return startPos, endPos - startPos, nil
}

// Улучшенный поиск последовательности span-ов с правильной нормализацией
func findBestSpanSequenceImproved(fragments []textFragment, normalizedTarget string) (int, int, error) {
	if len(fragments) == 0 {
		return -1, 0, nil
	}

	// Поиск лучшего совпадения с большим окном поиска
	bestStartIdx := -1
	bestEndIdx := -1
	bestSimilarity := 0.0

	// Скользящее окно по фрагментам
	for startIdx := 0; startIdx < len(fragments); startIdx++ {
		var combinedText strings.Builder
		
		for endIdx := startIdx; endIdx < len(fragments) && endIdx < startIdx+100; endIdx++ {
			combinedText.WriteString(fragments[endIdx].text)
			
			currentText := combinedText.String()
			currentNormalized := normalizeText(currentText) // Нормализуем полный текст
			
			// Проверяем различные типы совпадений
			exactMatch := strings.Contains(currentNormalized, normalizedTarget)
			similarity := calculateTextSimilarity(currentNormalized, normalizedTarget)
			
			// Проверяем частичные совпадения
			targetWords := strings.Fields(normalizedTarget)
			matchingWords := 0
			for _, word := range targetWords {
				if len(word) > 2 && strings.Contains(currentNormalized, word) {
					matchingWords++
				}
			}
			var partialMatch float64
			if len(targetWords) > 0 {
				partialMatch = float64(matchingWords)/float64(len(targetWords))
			}
			
			// Оценка качества совпадения
			score := similarity
			if exactMatch {
				score = 1.0 // Точное совпадение всегда лучше
			} else if partialMatch > 0.7 {
				score = partialMatch * 0.9 // Частичное совпадение тоже неплохо
			}
			
			if score > bestSimilarity && score > 0.6 {
				// Проверяем разумность длины
				startPos := fragments[startIdx].htmlPos
				endPos := fragments[endIdx].htmlPos + fragments[endIdx].htmlLen
				
				if endPos > startPos && endPos-startPos < len(normalizedTarget)*20 {
					bestSimilarity = score
					bestStartIdx = startIdx
					bestEndIdx = endIdx
					
					// Если нашли точное совпадение, можно остановиться
					if exactMatch {
						break
					}
				}
			}
			
			// Прекращаем, если текст стал слишком длинным
			if len(currentNormalized) > len(normalizedTarget)*5 {
				break
			}
		}
		
		// Если нашли точное совпадение, можно остановить поиск
		if bestSimilarity >= 1.0 {
			break
		}
	}

	if bestStartIdx == -1 || bestEndIdx == -1 {
		return -1, 0, nil
	}

	startPos := fragments[bestStartIdx].htmlPos
	endPos := fragments[bestEndIdx].htmlPos + fragments[bestEndIdx].htmlLen
	return startPos, endPos - startPos, nil
}

// Простая оценка схожести текста
func calculateTextSimilarity(text1, text2 string) float64 {
	if text1 == text2 {
		return 1.0
	}
	
	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	commonWords := 0
	for _, word1 := range words1 {
		for _, word2 := range words2 {
			if word1 == word2 && len(word1) > 2 {
				commonWords++
				break
			}
		}
	}
	
	maxLen := len(words1)
	if len(words2) > maxLen {
		maxLen = len(words2)
	}
	
	return float64(commonWords) / float64(maxLen)
}

// Проверка корректности найденного nested span match
func isNestedSpanMatch(candidateHTML, originalSubstr string) bool {
	// Извлекаем чистый текст из найденного HTML
	extractedText := extractTextFromHTML(candidateHTML)
	normalizedExtracted := normalizeText(extractedText)
	normalizedOriginal := normalizeText(originalSubstr)
	
	// Проверяем основные критерии
	if len(normalizedExtracted) == 0 || len(normalizedOriginal) == 0 {
		return false
	}
	
	// Не слишком длинный относительно оригинала
	if len(candidateHTML) > len(originalSubstr)*20 {
		return false
	}
	
	// Должен содержать большую часть оригинального текста или быть очень похожим
	similarity := calculateTextSimilarity(normalizedExtracted, normalizedOriginal)
	contains := strings.Contains(normalizedExtracted, normalizedOriginal)
	containsReverse := strings.Contains(normalizedOriginal, normalizedExtracted)
	
	return similarity > 0.7 || contains || containsReverse
}
