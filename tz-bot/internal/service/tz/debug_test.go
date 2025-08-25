package tzservice

import (
	"fmt"
	"strings"
	"testing"
)

func TestDebugSpanParsing(t *testing.T) {
	// Простой тест для отладки
	testHTML := `<p><span>Это</span> <span>тест</span> <span>множественных</span> <span>span</span></p>`
	target := "тест множественных span"
	
	fmt.Printf("=== ОТЛАДКА ===\n")
	fmt.Printf("HTML: %s\n", testHTML)
	fmt.Printf("Ищем: %s\n", target)
	fmt.Printf("Нормализованный целевой текст: '%s'\n", normalizeText(target))
	
	// Извлекаем и показываем чистый текст
	extractedText := extractTextFromHTML(testHTML)
	fmt.Printf("Извлеченный текст из HTML: '%s'\n", extractedText)
	fmt.Printf("Нормализованный извлеченный: '%s'\n", normalizeText(extractedText))
	
	// Проверяем содержание
	normalizedTarget := normalizeText(target)
	normalizedExtracted := normalizeText(extractedText)
	contains := containsSubstring(normalizedExtracted, normalizedTarget)
	fmt.Printf("Содержит целевой текст: %v\n", contains)
	
	// Тестируем findTextInSpanSequence
	position, length, err := findTextInSpanSequence(testHTML, normalizedTarget)
	fmt.Printf("findTextInSpanSequence: pos=%d, len=%d, err=%v\n", position, length, err)
	
	if position != -1 {
		candidateHTML := testHTML[position : position+length]
		fmt.Printf("Найденный HTML: '%s'\n", candidateHTML)
		
		isMatch := isNestedSpanMatch(candidateHTML, target)
		fmt.Printf("isNestedSpanMatch: %v\n", isMatch)
	}
	
	// Тестируем wrapInNestedSpans напрямую
	fmt.Printf("\n=== ТЕСТ wrapInNestedSpans ===\n")
	
	// Нужно извлечь содержимое тега p для тестирования
	blockContent := `<span>Это</span> <span>тест</span> <span>множественных</span> <span>span</span>`
	wrappedDirect, foundDirect, errDirect := wrapInNestedSpans(blockContent, target, "direct-test")
	fmt.Printf("wrapInNestedSpans прямой тест: found=%v, err=%v\n", foundDirect, errDirect)
	if foundDirect {
		fmt.Printf("Результат прямого теста: %s\n", wrappedDirect)
	}
	
	// Тестируем полный алгоритм
	result, found, err := WrapSubstringSmartHTML(testHTML, target, "debug-test")
	fmt.Printf("\n=== ПОЛНЫЙ РЕЗУЛЬТАТ ===\n")
	fmt.Printf("Найдено: %v, Ошибка: %v\n", found, err)
	if found {
		fmt.Printf("Результат: %s\n", result)
	}
	
	// Дополнительный тест: проверяем, что происходит в блочном поиске
	fmt.Printf("\n=== БЛОЧНЫЙ ПОИСК ===\n")
	
	// Тестируем компоненты findInBlocks для отладки
	blockTextContent := extractTextFromHTML(blockContent)
	blockNormalizedText := normalizeText(blockTextContent)
	blockNormalizedSubStr := normalizeText(target)
	
	fmt.Printf("Содержимое блока: '%s'\n", blockContent)
	fmt.Printf("Текст блока: '%s'\n", blockTextContent)  
	fmt.Printf("Нормализованный текст блока: '%s'\n", blockNormalizedText)
	fmt.Printf("Нормализованная подстрока: '%s'\n", blockNormalizedSubStr)
	
	// Проверяем точное содержание или частичное совпадение
	hasExactMatch := strings.Contains(blockNormalizedText, blockNormalizedSubStr)
	fmt.Printf("Точное совпадение: %v\n", hasExactMatch)
	
	if !hasExactMatch {
		// Проверяем частичное совпадение (большинство слов)
		subWords := strings.Fields(blockNormalizedSubStr)
		matchingWords := 0
		fmt.Printf("Слова подстроки: %v\n", subWords)
		for _, word := range subWords {
			if len(word) > 2 && strings.Contains(blockNormalizedText, word) {
				matchingWords++
				fmt.Printf("Найдено слово: '%s'\n", word)
			}
		}
		hasPartialMatch := len(subWords) > 0 && float64(matchingWords)/float64(len(subWords)) > 0.7
		fmt.Printf("Частичное совпадение: %v (%d из %d слов)\n", hasPartialMatch, matchingWords, len(subWords))
	}
	
	// Проверяем разумность размера блока
	reasonable := isBlockSizeReasonable(blockContent, target)
	fmt.Printf("Размер блока разумен: %v\n", reasonable)
	
	// Тестируем wrapWithinBlock напрямую
	fmt.Printf("\n=== ТЕСТ wrapWithinBlock ===\n")
	wrappedWithin, foundWithin, errWithin := wrapWithinBlock(blockContent, target, "within-test")
	fmt.Printf("wrapWithinBlock: found=%v, err=%v\n", foundWithin, errWithin)
	if foundWithin {
		fmt.Printf("Результат wrapWithinBlock: %s\n", wrappedWithin)
	}
	
	// Тестируем findInBlocks для тега 'p' напрямую
	wrappedP, foundP := findInBlocks(testHTML, target, "p-test", "p")
	fmt.Printf("findInBlocks для 'p': found=%v\n", foundP)
	if foundP {
		fmt.Printf("Результат поиска в p: %s\n", wrappedP)
	}
	
	wrapped, foundBlock, errBlock := findAndWrapMinimalBlock(testHTML, target, "block-test")
	fmt.Printf("findAndWrapMinimalBlock: found=%v, err=%v\n", foundBlock, errBlock)
	if foundBlock {
		fmt.Printf("Результат блочного поиска: %s\n", wrapped)
	}
}

func TestDebugComplexSpan(t *testing.T) {
	// Тестируем сложный пример из задачи
	testHTML := `<p style="margin: 0pt"><span style="white-space: pre-wrap">   Требования к режимам функционирования системы . </span><span lang="en-US" style="white-space: pre-wrap">MES</span><span style="white-space: pre-wrap">-система должна поддерживать основной режим, в котором выполняет все свои основные функции</span></p>`
	target := "Требования к режимам функционирования системы . MES-система должна поддерживать основной режим"
	
	fmt.Printf("\n=== ОТЛАДКА СЛОЖНОГО СЛУЧАЯ ===\n")
	fmt.Printf("HTML: %s\n", testHTML)
	fmt.Printf("Ищем: %s\n", target)
	
	// Извлекаем текст из HTML
	extractedText := extractTextFromHTML(testHTML)
	fmt.Printf("Извлеченный текст: '%s'\n", extractedText)
	fmt.Printf("Длина извлеченного: %d\n", len(extractedText))
	
	normalizedExtracted := normalizeText(extractedText)
	normalizedTarget := normalizeText(target)
	
	fmt.Printf("Нормализованный извлеченный: '%s'\n", normalizedExtracted)
	fmt.Printf("Нормализованный целевой: '%s'\n", normalizedTarget)
	
	// Проверяем, содержится ли текст
	contains := containsSubstring(normalizedExtracted, normalizedTarget)
	fmt.Printf("Содержит: %v\n", contains)
	
	// Проверяем размер блока
	reasonable := isBlockSizeReasonable(testHTML, target)
	fmt.Printf("Размер блока разумен: %v\n", reasonable)
	
	// Тестируем алгоритм
	result, found, err := WrapSubstringSmartHTML(testHTML, target, "debug-complex")
	fmt.Printf("Результат: found=%v, err=%v\n", found, err)
	if found {
		fmt.Printf("HTML результат: %s\n", result)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && stringContainsSubstring(s, substr)
}

func stringContainsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}