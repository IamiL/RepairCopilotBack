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
	wrapped := htmlStr[:start] +
		`<span error-id="` + html.EscapeString(id) + `">` +
		htmlStr[start:end] +
		`</span>` +
		htmlStr[end:]

	return wrapped, true, nil
}

// buildSpanningPattern строит regex, который позволяет встречаться HTML-тегам
// перед первым символом и между КАЖДЫМИ двумя символами подстроки.
// Пробельные символы в подстроке сворачиваются в (?:\s|&nbsp;)+.
func buildSpanningPattern(sub string) (string, error) {
	var b strings.Builder
	// Разрешаем сразу перед первым символом любой набор тегов
	b.WriteString(`(?:<[^>]*>)*`)

	rs := []rune(sub)
	for i, r := range rs {
		if unicode.IsSpace(r) {
			// Один "логический" пробел в подстроке соответствует любому кол-ву пробелов/переносов/&nbsp; в HTML
			b.WriteString(`(?:\s|&nbsp;)+`)
		} else {
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
		// Разрешаем теги между символами
		if i != len(rs)-1 {
			b.WriteString(`(?:<[^>]*>)*`)
		}
	}

	return b.String(), nil
}
