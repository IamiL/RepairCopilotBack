package tz

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
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
		highlightedPhrase := fmt.Sprintf(`<span error-id="%d">%s</span>`, id, escapedPhrase)

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

// extractErrorIds извлекает все error-id из span тегов в тексте
func ExtractErrorIds(text string) []string {
	// Регулярное выражение для поиска <span error-id="...">
	// Поддерживает пробелы вокруг атрибутов и другие атрибуты
	re := regexp.MustCompile(`<span[^>]*\berror-id="([^"]+)"[^>]*>`)

	// Найти все совпадения с группами захвата
	matches := re.FindAllStringSubmatch(text, -1)

	// Извлечь значения id из групп захвата
	var ids []string
	for _, match := range matches {
		if len(match) > 1 {
			ids = append(ids, match[1])
		}
	}

	return ids
}

// StringsToInts преобразует массив строк в массив int
// Возвращает ошибку, если какая-то строка не является числом
func StringsToInts(strings []string) ([]int, error) {
	ints := make([]int, len(strings))

	for i, str := range strings {
		num, err := strconv.Atoi(str)
		if err != nil {
			return nil, fmt.Errorf("не удалось преобразовать '%s' в число: %v", str, err)
		}
		ints[i] = num
	}

	return ints, nil
}
