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
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBlock))
	if err != nil {
		return nil, err
	}
	var out []string
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		tag := goquery.NodeName(s)
		if inlinePrefer[tag] {
			if strings.TrimSpace(s.Text()) != "" {
				if h, err := goquery.OuterHtml(s); err == nil {
					out = append(out, h)
				}
			}
		}
	})
	if len(out) == 0 {
		out = append(out, htmlBlock)
	}
	return out, nil
}
