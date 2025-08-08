package tzservice

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var blockContainers = map[string]bool{
	"ul": true, "ol": true, "table": true, "thead": true, "tbody": true, "tfoot": true,
	"tr": true, "div": true, "section": true, "article": true, "aside": true, "nav": true,
}

var inlinePrefer = map[string]bool{
	"li": true, "p": true, "span": true, "a": true, "em": true, "strong": true, "i": true, "b": true, "u": true, "small": true,
	"td": true, "th": true,
}

func isContainer(tag string) bool { return blockContainers[strings.ToLower(tag)] }

func collectLeafFragments(htmlBlock string) ([]string, error) {
	// Разбиваем контейнер на «листовые» фрагменты (outerHTML узлов, где реально есть текст).
	// Минимальный вариант: собрать outerHTML для каждого <li>, <p>, <span> и т.д. с текстом.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBlock))
	if err != nil {
		return nil, err
	}

	var out []string
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		tag := goquery.NodeName(s)
		if inlinePrefer[tag] {
			txt := strings.TrimSpace(s.Text())
			if txt != "" {
				// сохраним outerHTML
				if html, err := goquery.OuterHtml(s); err == nil {
					out = append(out, html)
				}
			}
		}
	})
	// Fallback: если ничего не нашли, вернём исходный блок как единственный фрагмент
	if len(out) == 0 {
		out = append(out, htmlBlock)
	}
	return out, nil
}
