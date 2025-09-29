package tzservice

import (
	"fmt"
	"regexp"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	"strings"

	"github.com/google/uuid"
)

//TODO: ЦИТАТ МНОГО, УЧИТЫВАЕТСЯ ТОЛЬКо ПЕВАЯ

func NewInvalidErrorsSet(startId uint32, report *[]tz_llm_client.GroupReport) (*[]OutInvalidError, uint32) {
	fmt.Println("НАЧИНАЕМ ФОРМИРОВАНИЕ NewInvalidErrorsSet")

	id := startId
	outInvalidErrors := make([]OutInvalidError, 0, 50)
	if report != nil {
		fmt.Println("report != nil")
		for i := range *report {
			if (*report)[i].Errors != nil {
				fmt.Println("(*report)[i].Errors != nil")
				for j := range *(*report)[i].Errors {
					if (*((*report)[i]).Errors)[j].Instances != nil {
						fmt.Println("(*report)[i].Errors[j].Instances != nil")
						for k := range *(*((*report)[i]).Errors)[j].Instances {
							if (*(*((*report)[i]).Errors)[j].Instances)[k].Kind != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].Kind == "Invalid" && (*(*((*report)[i]).Errors)[j].Instances)[k].Quotes != nil && (*(*((*report)[i]).Errors)[j].Instances)[k].Quotes[0] != "" {

								suggestedFix := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].Fix != nil {
									suggestedFix = *(*(*((*report)[i]).Errors)[j].Instances)[k].Fix
								}

								originalQuote := (*(*((*report)[i]).Errors)[j].Instances)[k].Quotes[0]

								cleanQuote := MarcdownCleaning((*(*((*report)[i]).Errors)[j].Instances)[k].Quotes[0])

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

								startLineNumber := (*(*((*report)[i]).Errors)[j].Instances)[k].Lines[0]
								//if startLineNumber == nil {
								//
								//}

								lengthLines := len((*(*((*report)[i]).Errors)[j].Instances)[k].Lines)
								endLineNumber := (*(*((*report)[i]).Errors)[j].Instances)[k].Lines[lengthLines-1]
								//if endLineNumber == nil {
								//
								//}

								var rationale string

								//if (*(*((*report)[i]).Errors)[j].Instances)[k].. != nil {
								//	rationale = *(*(*((*report)[i]).Errors)[j].Instances)[k].Rationale
								//}

								outInvalidErrors = append(outInvalidErrors, OutInvalidError{
									ID:                    uuid.New(),
									ErrorID:               (*((*report)[i]).Errors)[j].ID,
									HtmlID:                id,
									HtmlIDStr:             fmt.Sprintf("%d", id),
									Quote:                 cleanQuote,
									SuggestedFix:          suggestedFix,
									UntilTheEndOfSentence: EllipsisCheck((*(*((*report)[i]).Errors)[j].Instances)[k].Quotes[0]),
									StartLineNumber:       &startLineNumber,
									EndLineNumber:         &endLineNumber,
									QuoteLines:            quoteLines,
									OriginalQuote:         originalQuote,
									Rationale:             rationale,
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

	cleanStr = TrimPipesAndSpaces(cleanStr)

	cleanStr = RemoveMDBold(cleanStr)

	cleanStr = TrimEllipsis(cleanStr)

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

// TrimPipesAndSpaces удаляет вертикальные палки и пробелы с начала и конца строки
func TrimPipesAndSpaces(s string) string {
	return strings.Trim(s, "| ")
}

// TrimEllipsis удаляет троеточия с начала и конца строки
func TrimEllipsis(s string) string {
	// Убираем троеточия в начале
	for strings.HasPrefix(s, "...") {
		s = s[3:]
	}

	// Убираем троеточия в конце
	for strings.HasSuffix(s, "...") {
		s = s[:len(s)-3]
	}

	return s
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
