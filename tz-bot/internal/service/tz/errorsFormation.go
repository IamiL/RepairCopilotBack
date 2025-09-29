package tzservice

import (
	"fmt"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	promt_builder "repairCopilotBot/tz-bot/internal/pkg/promt-builder"
	"strconv"
)

func ErrorsFormation(groupReports []tz_llm_client.GroupReport, errorsDescriptions map[string]promt_builder.ErrorDescription) []Error {
	fmt.Println("ФОРМИРОВАНИЕ МАССИВА ОШИБОК---------------------------------------------")
	errors := make([]Error, 0)

	for i := range groupReports {
		groupID := ""
		if groupReports[i].GroupID == nil {
			continue
		}

		groupID = strconv.Itoa(*groupReports[i].GroupID)
		fmt.Println("ИЗ ГРУП АЙДИ ", *groupReports[i].GroupID, " ПОЛУЧИЛИ ", groupID)

		for j := range *groupReports[i].Errors {
			fmt.Println(strconv.Itoa(i) + ". ОБРАБОТКА ОШИБКИ:")
			fmt.Println((*groupReports[i].Errors)[j])
			errorCode := ""
			verdict := ""
			if (*groupReports[i].Errors)[j].Code == nil {
				fmt.Println("НЕ НАШЛИ ErrorCode")
				continue
			}
			//if (*groupReports[i].Errors)[j].Verdict == nil {
			//	continue
			//}

			errorCode = *(*groupReports[i].Errors)[j].Code
			verdict = (*groupReports[i].Errors)[j].Verdict.Status

			errorDescription := errorsDescriptions[errorCode]

			var processRetrieval *[]string

			//if (*groupReports[i].Errors)[j].AnalysisSteps.Retrieval != nil {
			//	processRetrievalArr := make([]string, 0)
			//
			//	for _, v := range *(*groupReports[i].Errors)[j].Process.Retrieval {
			//		if v.Text == nil {
			//			continue
			//		}
			//		processRetrievalArr = append(processRetrievalArr, *v.Text)
			//	}
			//
			//	processRetrieval = &processRetrievalArr
			//}

			newError := Error{
				ID:          (*groupReports[i].Errors)[j].ID,
				GroupID:     groupID,
				ErrorCode:   errorCode,
				Name:        *(*groupReports[i].Errors)[j].Title,
				Description: errorDescription.Desc,
				Detector:    errorDescription.Detector,
				//PreliminaryNotes:    groupReports[i].PreliminaryNotes,
				//OverallCritique:     groupReports[i].OverallCritique,
				Verdict: verdict,
				//ProcessAnalysis:     (*groupReports[i].Errors)[j].Process.Analysis,
				//ProcessCritique:     (*groupReports[i].Errors)[j].Process.Critique,
				//ProcessVerification: (*groupReports[i].Errors)[j].Process.Verification,
				ProcessRetrieval: processRetrieval,
				Instances:        (*groupReports[i].Errors)[j].Instances,
			}

			errors = append(errors, newError)

		}
	}

	return errors
}
