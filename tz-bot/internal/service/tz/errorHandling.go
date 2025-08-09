package tzservice

import (
	"fmt"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
)

type OutInvalidError struct {
	Id                    uint32
	IdStr                 string `json:"id"`
	GroupID               string `json:"group_id"`
	ErrorCode             string `json:"error_code"`
	Quote                 string `json:"quote"`
	Analysis              string `json:"analysis"`
	Critique              string `json:"critique"`
	Verification          string `json:"verification"`
	SuggestedFix          string `json:"suggested_fix"`
	Rationale             string `json:"rational"`
	UntilTheEndOfSentence bool
	StartLineNumber       *int
	EndLineNumber         *int
}

type OutMissingError struct {
	Id           uint32
	IdStr        string `json:"id"`
	GroupID      string `json:"group_id"`
	ErrorCode    string `json:"error_code"`
	Analysis     string `json:"analysis"`
	Critique     string `json:"critique"`
	Verification string `json:"verification"`
	SuggestedFix string `json:"suggested_fix"`
	Rationale    string `json:"rational"`
}

func HandleErrors(report *[]tz_llm_client.GroupReport, htmlBlocks *[]markdown_service_client.Mapping) (*[]OutInvalidError, *[]OutMissingError, string) {
	startId := uint32(0)

	outInvalidErrors, lastId := NewInvalidErrorsSet(startId, report)

	missingErrors, lastId := NewIMissingErrorsSet(lastId, report)

	errors := InjectInvalidErrorsToHtmlBlocks(outInvalidErrors, htmlBlocks)
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Println(err.Error())
		}
	}

	html := ""

	for i := range *htmlBlocks {
		html = html + (*htmlBlocks)[i].HtmlContent
	}

	return outInvalidErrors, missingErrors, html

}
