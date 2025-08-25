package tzservice

import (
	"fmt"
	"regexp"
	"testing"
)

func TestDebugFindInBlocks(t *testing.T) {
	testHTML := `<p><span>Это</span> <span>тест</span> <span>множественных</span> <span>span</span></p>`
	target := "тест множественных span"
	
	fmt.Printf("=== ОТЛАДКА findInBlocks ===\n")
	fmt.Printf("HTML: %s\n", testHTML)
	fmt.Printf("Target: %s\n", target)
	
	// Воспроизводим логику findInBlocks для тега 'p'
	tagName := "p"
	tagPattern := fmt.Sprintf(`(?is)(<(%s)\b[^>]*>)(.*?)(</%s>)`, regexp.QuoteMeta(tagName), regexp.QuoteMeta(tagName))
	fmt.Printf("Регулярное выражение: %s\n", tagPattern)
	
	re, err := regexp.Compile(tagPattern)
	if err != nil {
		t.Fatalf("Ошибка компиляции регекса: %v", err)
	}

	matches := re.FindAllStringSubmatchIndex(testHTML, -1)
	fmt.Printf("Найдено совпадений: %d\n", len(matches))
	
	if len(matches) == 0 {
		t.Fatalf("Не найдено блоков")
	}
	
	for i, loc := range matches {
		fmt.Printf("\n--- Блок %d ---\n", i+1)
		fmt.Printf("Индексы: %v\n", loc)
		
		if len(loc) < 10 {
			fmt.Printf("Недостаточно групп в совпадении\n")
			continue
		}

		openStart, openEnd := loc[2], loc[3]
		contentStart, contentEnd := loc[6], loc[7]
		closeStart, closeEnd := loc[8], loc[9]

		fullBlock := testHTML[loc[0]:loc[1]]
		openTag := testHTML[openStart:openEnd]
		blockContent := testHTML[contentStart:contentEnd]
		closeTag := testHTML[closeStart:closeEnd]
		
		fmt.Printf("Полный блок: '%s'\n", fullBlock)
		fmt.Printf("Открывающий тег: '%s'\n", openTag)
		fmt.Printf("Содержимое блока: '%s'\n", blockContent)
		fmt.Printf("Закрывающий тег: '%s'\n", closeTag)
		
		// Тестируем проверки
		textContent := extractTextFromHTML(blockContent)
		fmt.Printf("Текст из содержимого: '%s'\n", textContent)
		
		normalizedText := normalizeText(textContent)
		normalizedSubStr := normalizeText(target)
		fmt.Printf("Нормализованный текст: '%s'\n", normalizedText)
		fmt.Printf("Нормализованная подстрока: '%s'\n", normalizedSubStr)
		
		hasExactMatch := false // будет проверено ниже
		hasPartialMatch := false
		
		// Копируем точно ту же логику из findInBlocks
		if normalizedText != "" && normalizedSubStr != "" {
			hasExactMatch = containsSubstring(normalizedText, normalizedSubStr)
			fmt.Printf("Точное совпадение: %v\n", hasExactMatch)
			
			if !hasExactMatch {
				subWords := fieldsFromString(normalizedSubStr)
				matchingWords := 0
				for _, word := range subWords {
					if len(word) > 2 && containsSubstring(normalizedText, word) {
						matchingWords++
					}
				}
				hasPartialMatch = len(subWords) > 0 && float64(matchingWords)/float64(len(subWords)) > 0.7
				fmt.Printf("Частичное совпадение: %v (%d из %d слов)\n", hasPartialMatch, matchingWords, len(subWords))
			}
		}
		
		if !hasExactMatch && !hasPartialMatch {
			fmt.Printf("❌ Совпадение не найдено, продолжаем\n")
			continue
		}
		
		reasonable := isBlockSizeReasonable(blockContent, target)
		fmt.Printf("Размер блока разумен: %v\n", reasonable)
		
		if !reasonable {
			fmt.Printf("❌ Размер блока неразумен, продолжаем\n")
			continue
		}

		// Пытаемся обернуть внутри блока
		fmt.Printf("✅ Попытка оборачивания...\n")
		wrappedContent, found, wrapErr := wrapWithinBlock(blockContent, target, "test-id")
		fmt.Printf("wrapWithinBlock: found=%v, err=%v\n", found, wrapErr)
		
		if wrapErr != nil || !found {
			fmt.Printf("❌ Оборачивание не удалось\n")
			continue
		}
		
		fmt.Printf("✅ Успешно обернуто: %s\n", wrappedContent)
	}
}

func fieldsFromString(s string) []string {
	result := make([]string, 0)
	word := ""
	for _, r := range s {
		if r == ' ' {
			if word != "" {
				result = append(result, word)
				word = ""
			}
		} else {
			word += string(r)
		}
	}
	if word != "" {
		result = append(result, word)
	}
	return result
}