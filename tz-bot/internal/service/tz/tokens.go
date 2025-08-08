package tzservice

import (
	"regexp"
	"sort"
	"strings"
)

func tokensNormalized(s string) []string {
	// для сниппета
	s = stripMarkdown(s)
	s = normalizeText(s) // регистр/символы/единицы/кавычки
	re := regexp.MustCompile(`[\p{L}\p{N}]+|[.,:;!?()"-]`)
	return re.FindAllString(s, -1)
}

func informativeTokens(snippet string, maxTokens int) []string {
	toks := tokensNormalized(snippet)
	// оставляем только «словные» токены
	words := make([]string, 0, len(toks))
	for _, t := range toks {
		if regexp.MustCompile(`^[\p{L}\p{N}]{2,}$`).MatchString(t) {
			words = append(words, t)
		}
	}
	if len(words) == 0 {
		return toks
	}
	// берём самые длинные
	sort.SliceStable(words, func(i, j int) bool { return len([]rune(words[i])) > len([]rune(words[j])) })
	if len(words) > maxTokens {
		words = words[:maxTokens]
	}
	// сохранить порядок появления в сниппете
	order := map[string]int{}
	for i, t := range toks {
		if _, ok := order[t]; !ok {
			order[t] = i
		}
	}
	sort.SliceStable(words, func(i, j int) bool { return order[words[i]] < order[words[j]] })
	return words
}

func regexForTokens(toks []string) *regexp.Regexp {
	// вместо \b используем разделители: (^|[^[:alnum:]]) token ($|[^[:alnum:]])
	// между токенами позволяем «шум»: \P{L}\P{N}+
	var b strings.Builder
	b.WriteString(`(?s)`)
	for i, t := range toks {
		if i > 0 {
			b.WriteString(`[\P{L}\P{N}]+`)
		}
		b.WriteString(`(?:^|[^[:alnum:]])`)
		b.WriteString(`(` + regexp.QuoteMeta(t) + `)`)
		b.WriteString(`(?:$|[^[:alnum:]])`)
	}
	return regexp.MustCompile(b.String())
}
