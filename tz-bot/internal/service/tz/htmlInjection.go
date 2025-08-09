package tzservice

import (
	"fmt"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
)

func InjectInvalidErrorsToHtmlBlocks(invalidErrors *[]OutInvalidError, htmlBlocks *[]markdown_service_client.Mapping) []error {
	errors := make([]error, 0)
	for i := range *invalidErrors {
		if (*invalidErrors)[i].StartLineNumber != nil && (*invalidErrors)[i].EndLineNumber != nil {
			if *(*invalidErrors)[i].StartLineNumber == *(*invalidErrors)[i].EndLineNumber {
				err := injectIntoHTMLBlockByLineNumber(&(*invalidErrors)[i], htmlBlocks, *(*invalidErrors)[i].StartLineNumber)
				if err != nil {
					errors = append(errors, err)
				}
			} else {
				for line := *(*invalidErrors)[i].StartLineNumber; line <= *(*invalidErrors)[i].EndLineNumber; line++ {
					err := injectIntoHTMLBlockByLineNumber(&(*invalidErrors)[i], htmlBlocks, *(*invalidErrors)[i].StartLineNumber)
					if err != nil {
						errors = append(errors, err)
					}
				}
			}
		}
	}
	return errors
}

func injectIntoHTMLBlockByLineNumber(invalidError *OutInvalidError, htmlBlocks *[]markdown_service_client.Mapping, lineNumber int) error {
	for j := range *htmlBlocks {
		if (*htmlBlocks)[j].MarkdownStart <= lineNumber && (*htmlBlocks)[j].MarkdownEnd >= lineNumber {
			//newHtml := (*htmlBlocks)[j].HtmlContent
			newHtml, found := WrapSubstring((*htmlBlocks)[j].HtmlContent, invalidError.Quote, invalidError.IdStr)
			if found {
				(*htmlBlocks)[j].HtmlContent = newHtml
				return nil
			}

			newHtml, found, err := WrapSubstringApproxHTML(newHtml, invalidError.Quote, invalidError.IdStr)
			if err != nil {
				fmt.Println("Error in InjectInvalidErrorsToHtmlBlocks: ", err)
			}
			if found {
				(*htmlBlocks)[j].HtmlContent = newHtml
				return nil
			}

			fmt.Println("НЕ НАШЛИ СТРОКУ В HTML:")
			fmt.Println("HTML-блок:")
			fmt.Println((*htmlBlocks)[j].HtmlContent)
			fmt.Println("ПОДСТРОКА:")
			fmt.Println(invalidError.Quote)

			invalidError.GroupID = invalidError.GroupID + "НЕ НАЙДЕНО В ТЕКСТЕ"
			return fmt.Errorf("не нашли строку двумя способами")
		}
	}

	return fmt.Errorf("номер строки из md вышел за границы маппинга по html")
}
