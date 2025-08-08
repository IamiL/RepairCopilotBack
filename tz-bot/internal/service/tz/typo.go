package tzservice

import (
	"golang.org/x/net/html"
	"regexp"
	"strings"
)

// add near normalizeText
var confusables = map[rune]rune{
	// латиница <-> кириллица, самые болезненные
	'А': 'A', 'В': 'B', 'Е': 'E', 'К': 'K', 'М': 'M', 'Н': 'H', 'О': 'O', 'Р': 'P', 'С': 'C', 'Т': 'T', 'Х': 'X', 'У': 'Y',
	'a': 'a', 'е': 'e', 'к': 'k', 'м': 'm', 'н': 'h', 'o': 'o', 'р': 'p', 'с': 'c', 'т': 't', 'х': 'x', 'у': 'y',
	// обратные (лат->кир) не нужны — приводим всё к латинице
}

func normalizeSymbols(s string) string {
	// типографика и единицы
	var repl = map[string]string{
		// кавычки/многоточие/тире
		"“": "\"",
		"”": "\"",
		"„": "\"",
		"«": "\"",
		"»": "\"",
		"‘": "'",
		"’": "'",
		"…": "...",
		"—": "-",
		"–": "-",
		// пробелы
		"\u00A0": " ",
		"\u2009": " ",
		"\u202F": " ",
		// матем/единицы
		"×":  "x",
		"·":  "x",
		"˚":  "°",
		"″":  "\"", // дюймы
		"’’": "\"",
	}

	for k, v := range repl {
		s = strings.ReplaceAll(s, k, v)
	}

	// нормализация °C / ° C / градусы
	s = regexp.MustCompile(`\s*°\s*[CС]`).ReplaceAllString(s, "°C")         // C и кир. С
	s = regexp.MustCompile(`\b([0-9]+)\s*["”″]`).ReplaceAllString(s, `$1"`) // 55” -> 55"
	s = regexp.MustCompile(`\b4\s*[KК]\b`).ReplaceAllString(s, "4K")        // 4К -> 4K

	// Приведём похожие буквы к латинице (важно для 1С/1C, S/С и т.д.)
	var b strings.Builder
	for _, r := range s {
		if rr, ok := confusables[r]; ok {
			b.WriteRune(rr)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeText(s string) string {
	s = html.UnescapeString(s)
	s = normalizeSymbols(s)
	s = strings.ToLower(s) // регистр не важен
	s = strings.TrimSpace(s)
	reSpaces := regexp.MustCompile(`\s+`)
	s = reSpaces.ReplaceAllString(s, " ")
	return s
}

func stripMarkdown(s string) string {
	s = regexp.MustCompile("`([^`]*)`").ReplaceAllString(s, "$1")
	s = strings.NewReplacer("**", "", "__", "", "*", "", "_", "").Replace(s)
	s = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`(?m)^\s*[-*]\s+`).ReplaceAllString(s, "") // - bullets
	return s
}
