package tzservice

import (
	"fmt"
	"log/slog"
	"regexp"
	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	markdown_service_client "repairCopilotBot/tz-bot/internal/pkg/markdown-service"
	"strings"

	"github.com/google/uuid"
)

type OutInvalidError struct {
	ID                    uuid.UUID
	HtmlID                uint32
	HtmlIDStr             string
	ErrorID               uuid.UUID
	Rationale             string
	SuggestedFix          string
	Quote                 string
	OriginalQuote         string
	QuoteLines            *[]string
	UntilTheEndOfSentence bool
	StartLineNumber       *int
	EndLineNumber         *int
	SystemComment         string
	OrderNumber           int
	ParentError           Error
}

type OutMissingError struct {
	ID           uuid.UUID
	HtmlID       uint32
	HtmlIDStr    string
	ErrorID      uuid.UUID
	Rationale    string
	SuggestedFix string
}

func HandleErrors(report *[]tz_llm_client.GroupReport, htmlBlocks *[]markdown_service_client.Mapping) (*[]OutInvalidError, *[]OutMissingError, string) {
	startId := uint32(1)
	fmt.Println("отладка 21")
	outInvalidErrors, lastId := NewInvalidErrorsSet(startId, report)
	fmt.Println("отладка 22")
	missingErrors, lastId := NewIMissingErrorsSet(lastId, report)
	fmt.Println("отладка 23")
	errors := InjectInvalidErrorsToHtmlBlocks(outInvalidErrors, htmlBlocks)
	if len(errors) > 0 {
		fmt.Println("отладка 24")
		for _, err := range errors {
			fmt.Println(err.Error())
		}
	}
	fmt.Println("отладка 25")
	html := ""
	fmt.Println("отладка 26")

	for i := range *htmlBlocks {
		html = html + (*htmlBlocks)[i].HtmlContent
	}
	fmt.Println("отладка 27")

	// Сортируем ошибки по порядку их появления в HTML тексте
	sortedInvalidErrors := sortInvalidErrorsByHtmlOrder(outInvalidErrors, html)
	fmt.Println("отладка 28")
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
		errorMap[err.HtmlIDStr] = err
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
		if !addedIds[err.HtmlIDStr] {
			sortedErrors = append(sortedErrors, err)
		}
	}

	return &sortedErrors
}

// LogOutInvalidErrors красиво выводит в логи все OutInvalidErrors с детальной информацией
func LogOutInvalidErrors(log *slog.Logger, errors *[]OutInvalidError, prefix string) {
	if errors == nil {
		log.Info(prefix + " - OutInvalidErrors: nil")
		return
	}

	if len(*errors) == 0 {
		log.Info(prefix + " - OutInvalidErrors: пустой массив")
		return
	}

	log.Info(fmt.Sprintf("%s - OutInvalidErrors: найдено %d ошибок", prefix, len(*errors)))

	for i, err := range *errors {
		// Основная информация об ошибке
		//log.Info(fmt.Sprintf("  [%d] Ошибка ID=%d, IdStr=%s", i+1, err.Id, err.IdStr),
		//	slog.String("group_id", err.GroupID),
		//	slog.String("error_code", err.ErrorCode))

		// Цитата и оригинальная цитата
		if err.Quote != "" {
			log.Info(fmt.Sprintf("    Quote: %s", truncateString(err.Quote, 100)))
		}
		if err.OriginalQuote != "" {
			log.Info(fmt.Sprintf("    OriginalQuote: %s", truncateString(err.OriginalQuote, 100)))
		}

		// QuoteLines (указатель на массив строк)
		if err.QuoteLines != nil {
			quoteLinesSlice := *err.QuoteLines
			log.Info(fmt.Sprintf("    QuoteLines (%d строк):", len(quoteLinesSlice)))
			for j, line := range quoteLinesSlice {
				log.Info(fmt.Sprintf("      [%d]: %s", j+1, truncateString(line, 80)))
			}
		} else {
			log.Info("    QuoteLines: nil")
		}

		// Номера строк (указатели на int)
		startLineInfo := "nil"
		if err.StartLineNumber != nil {
			startLineInfo = fmt.Sprintf("%d", *err.StartLineNumber)
		}

		endLineInfo := "nil"
		if err.EndLineNumber != nil {
			endLineInfo = fmt.Sprintf("%d", *err.EndLineNumber)
		}

		log.Info(fmt.Sprintf("    Строки: %s - %s", startLineInfo, endLineInfo),
			slog.Bool("until_end_of_sentence", err.UntilTheEndOfSentence))

		// Анализ и рекомендации
		//if err.Analysis != "" {
		//	log.Info(fmt.Sprintf("    Analysis: %s", truncateString(err.Analysis, 150)))
		//}
		//if err.Critique != "" {
		//	log.Info(fmt.Sprintf("    Critique: %s", truncateString(err.Critique, 150)))
		//}
		//if err.Verification != "" {
		//	log.Info(fmt.Sprintf("    Verification: %s", truncateString(err.Verification, 150)))
		//}
		//if err.SuggestedFix != "" {
		//	log.Info(fmt.Sprintf("    SuggestedFix: %s", truncateString(err.SuggestedFix, 150)))
		//}
		//if err.Rationale != "" {
		//	log.Info(fmt.Sprintf("    Rationale: %s", truncateString(err.Rationale, 150)))
		//}

		// Разделитель между ошибками
		if i < len(*errors)-1 {
			log.Info("    " + strings.Repeat("-", 50))
		}
	}
}

// truncateString обрезает строку до указанной длины, добавляя "..." если строка была обрезана
func truncateString(s string, maxLen int) string {
	// Убираем переносы строк и лишние пробелы для лучшего отображения в логах
	cleaned := strings.ReplaceAll(s, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	re := regexp.MustCompile(`\s+`)
	cleaned = re.ReplaceAllString(strings.TrimSpace(cleaned), " ")

	if len(cleaned) <= maxLen {
		return cleaned
	}

	return cleaned[:maxLen-3] + "..."
}
