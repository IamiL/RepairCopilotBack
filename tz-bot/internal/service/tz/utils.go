package tzservice

import (
	"regexp"
	"sort"
	"strings"
)

func informativeTokens(snippet string, maxTokens int) []string {
	toks := tokens(snippet) // уже stripMarkdown + normalizeText внутри tokens
	sort.SliceStable(toks, func(i, j int) bool { return len(toks[i]) > len(toks[j]) })
	// убираем чистую пунктуацию
	out := make([]string, 0, len(toks))
	for _, t := range toks {
		if regexp.MustCompile(`^[\p{L}\p{N}]{2,}$`).MatchString(t) {
			out = append(out, t)
		}
		if len(out) == maxTokens {
			break
		}
	}
	if len(out) == 0 {
		return tokens(snippet)
	}
	// сохраняем исходный порядок inform-токенов, как в сниппете
	order := map[string]int{}
	for i, tok := range tokens(snippet) {
		if _, ok := order[tok]; !ok {
			order[tok] = i
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return order[out[i]] < order[out[j]] })
	return out
}

func regexForTokens(toks []string) *regexp.Regexp {
	// \b tok1 \b (?:\W+|\s+)* \b tok2 \b ...
	// где \W допускает пунктуацию; уже в normalizeText всё в нижнем регистре
	var b strings.Builder
	b.WriteString(`(?s)`)
	for i, t := range toks {
		if i > 0 {
			b.WriteString(`(?:\W+|\s+)*`)
		}
		b.WriteString(`\b` + regexp.QuoteMeta(t) + `\b`)
	}
	return regexp.MustCompile(b.String())
}

func findBestMatch(snippet, plain string) match {
	// 1) полный набор токенов
	toksFull := tokens(snippet)
	if len(toksFull) == 0 || len(plain) == 0 {
		return match{}
	}

	// 1a) сначала пробуем информативное ядро (чтобы не рвать на тегах)
	toks := informativeTokens(snippet, 6) // 6-8 обычно хватает
	if len(toks) >= 2 {
		re := regexForTokens(toks)
		if loc := re.FindStringIndex(plain); loc != nil {
			return match{Start: loc[0], End: loc[1], Score: loc[1] - loc[0], Found: true}
		}
	}

	// 2) попытка полным списком токенов
	re2 := regexForTokens(toksFull)
	if loc := re2.FindStringIndex(plain); loc != nil {
		return match{Start: loc[0], End: loc[1], Score: loc[1] - loc[0], Found: true}
	}

	// 3) «укороченный сниппет»: если сниппет оканчивается на ... — ищем как префикс
	if strings.HasSuffix(strings.TrimSpace(stripMarkdown(snippet)), "...") {
		base := strings.TrimSuffix(tokens(snippet)[0], "...")
		if base != "" {
			// возьмём первые 3 информативных токена
			toks3 := informativeTokens(snippet, 3)
			re3 := regexForTokens(toks3)
			if loc := re3.FindStringIndex(plain); loc != nil {
				return match{Start: loc[0], End: loc[1], Score: loc[1] - loc[0], Found: true}
			}
		}
	}

	// 4) fallback: скользящее окно по 3-4 токена
	baseToks := tokens(snippet)
	win := 4
	if len(baseToks) < win {
		win = len(baseToks)
	}
	for i := 0; i+win <= len(baseToks); i++ {
		re := regexForTokens(baseToks[i : i+win])
		if loc := re.FindStringIndex(plain); loc != nil {
			return match{Start: loc[0], End: loc[1], Score: loc[1] - loc[0], Found: true}
		}
	}

	return match{}
}

// add near normalizeText
//var confusables = map[rune]rune{
//	// латиница <-> кириллица, самые болезненные
//	'А': 'A', 'В': 'B', 'Е': 'E', 'К': 'K', 'М': 'M', 'Н': 'H', 'О': 'O', 'Р': 'P', 'С': 'C', 'Т': 'T', 'Х': 'X', 'У': 'Y',
//	'a': 'a', 'е': 'e', 'к': 'k', 'м': 'm', 'н': 'h', 'o': 'o', 'р': 'p', 'с': 'c', 'т': 't', 'х': 'x', 'у': 'y',
//	// обратные (лат->кир) не нужны — приводим всё к латинице
//}
//
//func normalizeSymbols(s string) string {
//	// типографика и единицы
//	repl := map[string]string{
//		// кавычки/многоточие/тире
//		"“": "\"", "”": "\"", "„": "\"", "«": "\"", "»": "\"", "‘": "'", "’": "'", "…": "...",
//		"—": "-", "–": "-",
//		// пробелы
//		"\u00A0": " ", "\u2009": " ", "\u202F": " ",
//		// матем/единицы
//		"×": "x", "·": "x", "˚": "°",
//		"″": "\"", "’’": "\"", "”": "\"", // иногда дюймы/секунды угла
//	}
//	for k, v := range repl {
//		s = strings.ReplaceAll(s, k, v)
//	}
//
//	// нормализация °C / ° C / градусы
//	s = regexp.MustCompile(`\s*°\s*[CС]`).ReplaceAllString(s, "°C")         // C и кир. С
//	s = regexp.MustCompile(`\b([0-9]+)\s*["”″]`).ReplaceAllString(s, `$1"`) // 55” -> 55"
//	s = regexp.MustCompile(`\b4\s*[KК]\b`).ReplaceAllString(s, "4K")        // 4К -> 4K
//
//	// Приведём похожие буквы к латинице (важно для 1С/1C, S/С и т.д.)
//	var b strings.Builder
//	for _, r := range s {
//		if rr, ok := confusables[r]; ok {
//			b.WriteRune(rr)
//		} else {
//			b.WriteRune(r)
//		}
//	}
//	return b.String()
//}
//
//func normalizeText(s string) string {
//	s = html.UnescapeString(s)
//	s = normalizeSymbols(s)
//	s = strings.ToLower(s) // регистр не важен
//	s = strings.TrimSpace(s)
//	reSpaces := regexp.MustCompile(`\s+`)
//	s = reSpaces.ReplaceAllString(s, " ")
//	return s
//}
//
//func stripMarkdown(s string) string {
//	s = regexp.MustCompile("`([^`]*)`").ReplaceAllString(s, "$1")
//	s = strings.NewReplacer("**", "", "__", "", "*", "", "_", "").Replace(s)
//	s = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(s, "$1")
//	s = regexp.MustCompile(`(?m)^\s*[-*]\s+`).ReplaceAllString(s, "") // - bullets
//	return s
//}
