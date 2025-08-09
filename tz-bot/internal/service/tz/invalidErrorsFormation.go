package tzservice

import (
	"fmt"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	"strings"
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
							if (*(*((*report)[i]).Errors)[j].Instances)[k].ErrType != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].ErrType == "invalid" && *(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet != "" {
								groupId := ""
								if (*report)[i].GroupID != nil {
									groupId = *(*report)[i].GroupID
								}
								errorCode := ""
								if (*((*report)[i]).Errors)[j].Code != nil {
									errorCode = *(*((*report)[i]).Errors)[j].Code
								}
								analysis := ""
								critique := ""
								verification := ""
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
								}

								suggested_fix := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix != nil {
									suggested_fix = *(*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix
								}

								rationale := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].Rationale != nil {
									rationale = *(*(*((*report)[i]).Errors)[j].Instances)[k].Rationale
								}

								cleanQuote := MarcdownCleaning(*(*(*((*report)[i]).Errors)[j].Instances)[k].Snippet)

								startLineNumber := (*(*((*report)[i]).Errors)[j].Instances)[k].LineStart
								//if startLineNumber == nil {
								//
								//}
								endLineNumber := (*(*((*report)[i]).Errors)[j].Instances)[k].LineEnd
								//if endLineNumber == nil {
								//
								//}

								outInvalidErrors = append(outInvalidErrors, OutInvalidError{
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
	return cleanStr
}
