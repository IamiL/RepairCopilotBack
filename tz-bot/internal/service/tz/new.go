package tz

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

type Tz struct {
}

// HighlightPhraseIgnoreCase ищет фразу без учета регистра
func HighlightPhraseIgnoreCase(text, phrase string, id int) string {
	if phrase == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerPhrase := strings.ToLower(phrase)

	// Находим позицию фразы (без учета регистра)
	index := strings.Index(lowerText, lowerPhrase)
	if index == -1 {
		return text // Фраза не найдена
	}

	result := text
	for index != -1 {
		// Извлекаем оригинальную фразу с сохранением регистра
		originalPhrase := result[index : index+len(phrase)]
		escapedPhrase := html.EscapeString(originalPhrase)
		highlightedPhrase := fmt.Sprintf(`<span data-error="%d" class="error-text">%s</span>`, id, escapedPhrase)

		// Заменяем найденную фразу
		result = result[:index] + highlightedPhrase + result[index+len(phrase):]

		// Ищем следующее вхождение
		searchStart := index + len(highlightedPhrase)
		if searchStart >= len(result) {
			break
		}

		lowerResult := strings.ToLower(result[searchStart:])
		nextIndex := strings.Index(lowerResult, lowerPhrase)
		if nextIndex == -1 {
			break
		}
		index = searchStart + nextIndex
	}

	return result
}

func FixHTMLTags(input string) string {
	// Регулярное выражение для открывающих тегов <p[числа]>
	openTagRegex := regexp.MustCompile(`<p\d+>`)

	// Регулярное выражение для закрывающих тегов </p[числа]>
	closeTagRegex := regexp.MustCompile(`</p\d+>`)

	// Заменяем открывающие теги
	result := openTagRegex.ReplaceAllString(input, "<p>")

	// Заменяем закрывающие теги
	result = closeTagRegex.ReplaceAllString(result, "</p>")

	return result
}
