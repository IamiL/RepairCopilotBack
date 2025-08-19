package tzservice

import tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"

func ErrorsFormation(groupReports []tz_llm_client.GroupReport) []Error {
	errors := make([]Error, 0)

	for i := range groupReports {
		groupID := ""
		if groupReports[i].GroupID == nil {
			continue
		}

		groupID = *groupReports[i].GroupID

		for j := range *groupReports[i].Errors {
			errorCode := ""
			verdict := ""
			if (*groupReports[i].Errors)[j].Code == nil {
				continue
			}
			if (*groupReports[i].Errors)[j].Verdict == nil {
				continue
			}

			errorCode = *(*groupReports[i].Errors)[j].Code
			verdict = *(*groupReports[i].Errors)[j].Verdict

			var processRetrieval *[]string

			if (*groupReports[i].Errors)[j].Process.Retrieval != nil {
				processRetrievalArr := make([]string, 0)

				for _, v := range *(*groupReports[i].Errors)[j].Process.Retrieval {
					if v.Text == nil {
						continue
					}
					processRetrievalArr = append(processRetrievalArr, *v.Text)
				}

				processRetrieval = &processRetrievalArr
			}

			newError := Error{
				ID:                  (*groupReports[i].Errors)[j].ID,
				GroupID:             groupID,
				ErrorCode:           errorCode,
				PreliminaryNotes:    groupReports[i].PreliminaryNotes,
				OverallCritique:     groupReports[i].OverallCritique,
				Verdict:             verdict,
				ProcessAnalysis:     (*groupReports[i].Errors)[j].Process.Analysis,
				ProcessCritique:     (*groupReports[i].Errors)[j].Process.Critique,
				ProcessVerification: (*groupReports[i].Errors)[j].Process.Verification,
				ProcessRetrieval:    processRetrieval,
				Instances:           (*groupReports[i].Errors)[j].Instances,
			}

			errors = append(errors, newError)

		}
	}

	return errors
}
