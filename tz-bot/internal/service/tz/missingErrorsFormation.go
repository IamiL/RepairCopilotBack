package tzservice

import (
	"fmt"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
)

func NewIMissingErrorsSet(startId uint32, report *[]tz_llm_client.GroupReport) (*[]OutMissingError, uint32) {
	id := startId
	outInvalidErrors := make([]OutMissingError, 0, 50)
	if report != nil {
		for i := range *report {
			if (*report)[i].Errors != nil {
				for j := range *(*report)[i].Errors {
					if (*((*report)[i]).Errors)[j].Instances != nil {
						for k := range *(*((*report)[i]).Errors)[j].Instances {
							if (*(*((*report)[i]).Errors)[j].Instances)[k].ErrType != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].ErrType == "missing" {
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

								suggestedFix := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix != nil {
									suggestedFix = *(*(*((*report)[i]).Errors)[j].Instances)[k].SuggestedFix
								}

								rationale := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].Rationale != nil {
									rationale = *(*(*((*report)[i]).Errors)[j].Instances)[k].Rationale
								}

								outInvalidErrors = append(outInvalidErrors, OutMissingError{
									Id:           id,
									IdStr:        fmt.Sprintf("%d", id),
									GroupID:      groupId,
									ErrorCode:    errorCode,
									Analysis:     analysis,
									Critique:     critique,
									Verification: verification,
									SuggestedFix: suggestedFix,
									Rationale:    rationale,
									Retrieval:    retrieval,
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
