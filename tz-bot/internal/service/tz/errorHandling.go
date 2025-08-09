package tzservice

import (
	"fmt"
	"regexp"
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

	// Сортируем ошибки по порядку их появления в HTML тексте
	sortedInvalidErrors := sortInvalidErrorsByHtmlOrder(outInvalidErrors, html)

	return sortedInvalidErrors, missingErrors, html
}

// extractErrorIdsFromHtml извлекает error-id из span тегов в HTML в порядке их появления
func extractErrorIdsFromHtml(htmlText string) []string {
	// Регулярное выражение для поиска <span error-id="...">
	// Поддерживает различные форматы атрибутов и пробелы
	re := regexp.MustCompile(`<span[^>]*\berror-id=["']([^"']+)["'][^>]*>`)
	
	matches := re.FindAllStringSubmatch(htmlText, -1)
	
	var errorIds []string
	for _, match := range matches {
		if len(match) > 1 {
			errorIds = append(errorIds, match[1])
		}
	}
	
	return errorIds
}

// sortInvalidErrorsByHtmlOrder сортирует массив ошибок по порядку их появления в HTML
func sortInvalidErrorsByHtmlOrder(errors *[]OutInvalidError, htmlText string) *[]OutInvalidError {
	if errors == nil || len(*errors) == 0 {
		return errors
	}
	
	// Извлекаем порядок ID из HTML
	htmlErrorIds := extractErrorIdsFromHtml(htmlText)
	if len(htmlErrorIds) == 0 {
		return errors // Возвращаем исходный порядок, если ID не найдены
	}
	
	// Создаем map для быстрого поиска ошибок по IdStr
	errorMap := make(map[string]OutInvalidError)
	for _, err := range *errors {
		errorMap[err.IdStr] = err
	}
	
	// Создаем отсортированный массив
	var sortedErrors []OutInvalidError
	addedIds := make(map[string]bool)
	
	// Добавляем ошибки в порядке появления в HTML
	for _, htmlId := range htmlErrorIds {
		if err, exists := errorMap[htmlId]; exists && !addedIds[htmlId] {
			sortedErrors = append(sortedErrors, err)
			addedIds[htmlId] = true
		}
	}
	
	// Добавляем оставшиеся ошибки, которых нет в HTML (на случай если что-то пропустили)
	for _, err := range *errors {
		if !addedIds[err.IdStr] {
			sortedErrors = append(sortedErrors, err)
		}
	}
	
	return &sortedErrors
}
