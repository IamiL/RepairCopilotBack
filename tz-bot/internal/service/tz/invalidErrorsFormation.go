package tzservice

import (
	"fmt"
	"regexp"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	"strings"

	"github.com/google/uuid"
)

func NewInvalidErrorsSet(startId uint32, report *[]tz_llm_client.GroupReport) (*[]OutInvalidError, uint32) {
	id := startId
	outInvalidErrors := make([]OutInvalidError, 0, 50)
	if report != nil {
		for i := range *report {
			if (*report)[i].Errors != nil {
				for j := range *(*report)[i].Errors {
					if (*((*report)[i]).Errors)[j].Instances != nil {
						for k := range *(*((*report)[i]).Errors)[j].Instances {
							if (*(*((*report)[i]).Errors)[j].Instances)[k].ErrType != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].ErrType == "invalid" && (*(*((*report)[i]).Errors)[j].Instances)[k].Snippet != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet != "" {

								suggestedFix := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix != nil {
									suggestedFix = *(*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix
								}
								originalQuote := *(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet

								cleanQuote := MarcdownCleaning(*(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet)

								var quoteLines *[]string

								cleanQuoteLines := SplitLinesNoEmpty(cleanQuote)

								if cleanQuoteLines != nil && len(cleanQuoteLines) > 1 {
									for lineNimber, _ := range cleanQuoteLines {
										cleanQuoteLines[lineNimber] = MarcdownCleaning(cleanQuoteLines[lineNimber])
									}

									quoteLines = &cleanQuoteLines

								} else {
									cleanQuoteCells := SplitByPipeNoEmpty(cleanQuote)

									if cleanQuoteCells != nil && len(cleanQuoteCells) > 1 {
										for lineNimber, _ := range cleanQuoteCells {
											cleanQuoteCells[lineNimber] = MarcdownCleaning(cleanQuoteCells[lineNimber])
										}
										quoteLines = &cleanQuoteCells
									}
								}

								startLineNumber := (*(*((*report)[i]).Errors)[j].Instances)[k].LineStart
								//if startLineNumber == nil {
								//
								//}
								endLineNumber := (*(*((*report)[i]).Errors)[j].Instances)[k].LineEnd
								//if endLineNumber == nil {
								//
								//}

								outInvalidErrors = append(outInvalidErrors, OutInvalidError{
									ID:                    uuid.New(),
									ErrorID:               (*((*report)[i]).Errors)[j].ID,
									HtmlID:                id,
									HtmlIDStr:             fmt.Sprintf("%d", id),
									Quote:                 cleanQuote,
									SuggestedFix:          suggestedFix,
									UntilTheEndOfSentence: EllipsisCheck(*(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet),
									StartLineNumber:       startLineNumber,
									EndLineNumber:         endLineNumber,
									QuoteLines:            quoteLines,
									OriginalQuote:         originalQuote,
								})

								id++
							}
						}
					}

				}
			}

		}
	}

	return &outInvalidErrors, id
}

func EllipsisCheck(str string) bool {
	return strings.HasSuffix(str, "...")
}

func MarcdownCleaning(markdown string) string {
	cleanStr := markdown
	if strings.HasPrefix(markdown, "- ") {
		cleanStr = cleanStr[2:]
	}

	//удаление префиксов `[0123124] - `
	// RemoveBracketPrefix удаляет префикс вида "[что-то] - " в начале строки
	var prefixRegex1 = regexp.MustCompile(`\[[^\]]*\] - `)
	cleanStr = prefixRegex1.ReplaceAllString(cleanStr, "")

	// удаление префиксов `[234525] `
	cleanStr, _ = TrimBracketPrefix(cleanStr)

	// удаление префиксов `## `
	if strings.HasPrefix(cleanStr, "## ") {
		cleanStr = cleanStr[3:]
	}

	cleanStr = RemoveMDBold(cleanStr)

	return cleanStr
}

func RemoveMDBold(s string) string {
	// ***bold+italic***
	reTriple := regexp.MustCompile(`(?s)\*{3}(.+?)\*{3}`)
	// **bold**
	reDoubleAsterisk := regexp.MustCompile(`(?s)\*{2}(.+?)\*{2}`)
	// __bold__
	reDoubleUnderscore := regexp.MustCompile(`(?s)_{2}(.+?)_{2}`)

	out := reTriple.ReplaceAllString(s, `$1`)
	out = reDoubleAsterisk.ReplaceAllString(out, `$1`)
	out = reDoubleUnderscore.ReplaceAllString(out, `$1`)
	return out
}

func TrimBracketPrefix(s string) (string, bool) {
	// Минимум: "[" + 1 символ + "]" + " " => длина >= 4
	if len(s) < 4 || s[0] != '[' {
		return s, false
	}

	// Ищем первую закрывающую скобку
	i := strings.IndexByte(s, ']')
	// i==1 -> внутри скобок ничего (нарушает "несколько символов")
	if i <= 1 {
		return s, false
	}

	// Проверяем, что сразу после ']' стоит пробел
	if i+1 < len(s) && s[i+1] == ' ' {
		return s[i+2:], true
	}

	return s, false
}

func SplitLinesNoEmpty(s string) []string {
	rawLines := strings.Split(s, "\n")
	var lines []string
	for _, line := range rawLines {
		line = strings.TrimSpace(line) // убираем пробелы и \r
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func SplitByPipeNoEmpty(s string) []string {
	rawParts := strings.Split(s, " | ")
	var parts []string
	for _, p := range rawParts {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
