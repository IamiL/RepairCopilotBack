package tzservice

import (
	"fmt"
	"regexp"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	"strings"
)

func NewInvalidErrorsSet(startId uint32, report *[]tz_llm_client.GroupReport) (*[]OutInvalidError, uint32) {
	id := startId
	fmt.Println("отладка 71")
	outInvalidErrors := make([]OutInvalidError, 0, 50)
	fmt.Println("отладка 72")
	if report != nil {
		fmt.Println("отладка 73")
		for i := range *report {
			fmt.Println("отладка 74")
			if (*report)[i].Errors != nil {
				fmt.Println("отладка 75")
				for j := range *(*report)[i].Errors {
					fmt.Println("отладка 76")
					if (*((*report)[i]).Errors)[j].Instances != nil {
						fmt.Println("отладка 77")
						for k := range *(*((*report)[i]).Errors)[j].Instances {
							fmt.Println("отладка 78")
							if (*(*((*report)[i]).Errors)[j].Instances)[k].ErrType != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].ErrType == "invalid" && (*(*((*report)[i]).Errors)[j].Instances)[k].Snippet != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet != "" {
								fmt.Println("отладка 31")
								groupId := ""
								if (*report)[i].GroupID != nil {
									groupId = *(*report)[i].GroupID
								}
								errorCode := ""
								fmt.Println("отладка 32")
								if (*((*report)[i]).Errors)[j].Code != nil {
									errorCode = *(*((*report)[i]).Errors)[j].Code
								}
								fmt.Println("отладка 33")
								analysis := ""
								critique := ""
								verification := ""
								var retrieval []string
								if (*((*report)[i]).Errors)[j].Process != nil {
									if (*((*report)[i]).Errors)[j].Process.Analysis != nil {
										analysis = *(*((*report)[i]).Errors)[j].Process.Analysis
									}
									if (*((*report)[i]).Errors)[j].Process.Critique != nil {
										critique = *(*((*report)[i]).Errors)[j].Process.Critique
									}
									if (*((*report)[i]).Errors)[j].Process.Verification != nil {
										verification = *(*((*report)[i]).Errors)[j].Process.Verification
									}
									// Извлекаем тексты из Retrieval
									if (*((*report)[i]).Errors)[j].Process.Retrieval != nil {
										for _, r := range *(*((*report)[i]).Errors)[j].Process.Retrieval {
											if r.Text != nil {
												retrieval = append(retrieval, *r.Text)
											}
										}
									}
								}
								fmt.Println("отладка 34")
								suggested_fix := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix != nil {
									suggested_fix = *(*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix
								}
								fmt.Println("отладка 35")
								rationale := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].Rationale != nil {
									rationale = *(*(*((*report)[i]).Errors)[j].Instances)[k].Rationale
								}
								fmt.Println("отладка 36")
								originalQuote := *(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet

								fmt.Println("отладка 37")
								cleanQuote := MarcdownCleaning(*(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet)

								fmt.Println("отладка 38")
								var quoteLines *[]string

								fmt.Println("отладка 39")
								cleanQuoteLines := SplitLinesNoEmpty(cleanQuote)

								fmt.Println("отладка 40")
								if cleanQuoteLines != nil && len(cleanQuoteLines) > 1 {
									fmt.Println("отладка 41")
									for lineNimber, _ := range cleanQuoteLines {
										cleanQuoteLines[lineNimber] = MarcdownCleaning(cleanQuoteLines[lineNimber])
									}
									fmt.Println("отладка 42")

									quoteLines = &cleanQuoteLines

								} else {
									fmt.Println("отладка 43")
									cleanQuoteCells := SplitByPipeNoEmpty(cleanQuote)
									fmt.Println("отладка 44")
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
									ErrorID:               (*((*report)[i]).Errors)[j].ID,
									Id:                    id,
									IdStr:                 fmt.Sprintf("%d", id),
									GroupID:               groupId,
									ErrorCode:             errorCode,
									Quote:                 cleanQuote,
									Analysis:              analysis,
									Critique:              critique,
									Verification:          verification,
									SuggestedFix:          suggested_fix,
									Rationale:             rationale,
									UntilTheEndOfSentence: EllipsisCheck(*(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet),
									StartLineNumber:       startLineNumber,
									EndLineNumber:         endLineNumber,
									QuoteLines:            quoteLines,
									OriginalQuote:         originalQuote,
									Retrieval:             retrieval,
								})

								id++
							}
						}
					}

				}
			}

		}
	}
	fmt.Println("отладка 79")

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
