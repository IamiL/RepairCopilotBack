package tzservice

import (
	"fmt"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"

	"github.com/google/uuid"
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
							if (*(*((*report)[i]).Errors)[j].Instances)[k].Kind != nil && *(*(*((*report)[i]).Errors)[j].Instances)[k].Kind == "Missing" {
								suggestedFix := ""
								if (*(*((*report)[i]).Errors)[j].Instances)[k].Fix != nil {
									suggestedFix = *(*(*((*report)[i]).Errors)[j].Instances)[k].Fix
								}

								var rationale string
								//if (*(*((*report)[i]).Errors)[j].Instances)[k].Rationale != nil {
								//	rationale = *(*(*((*report)[i]).Errors)[j].Instances)[k].Rationale
								//}

								outInvalidErrors = append(outInvalidErrors, OutMissingError{
									ErrorID:      (*((*report)[i]).Errors)[j].ID,
									HtmlID:       id,
									HtmlIDStr:    fmt.Sprintf("%d", id),
									SuggestedFix: suggestedFix,
									Rationale:    rationale,
									ID:           uuid.New(),
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
