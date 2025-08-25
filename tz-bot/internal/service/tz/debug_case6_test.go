package tzservice

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestDebugCase6(t *testing.T) {
	html := `<p data-mapping-id="27e5ccae-cc60-482e-a7f4-aa79e0eefb19" style="margin: 0pt; min-height: 1em; margin-top: 5pt; margin-bottom: 5pt; line-height: 1; text-align: justify;"><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>3</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>.</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>2</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'> </span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>Требование к ролям и полномочиям</span></p>`
	target := `3.2 Требование к ролям и полномочиям`
	
	fmt.Printf("=== ОТЛАДКА CASE 6 ===\n")
	fmt.Printf("HTML: %s\n", html)
	fmt.Printf("Target: %s\n", target)
	
	// Извлекаем содержимое p-тега
	blockContent := `<span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>3</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>.</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>2</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'> </span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>Требование к ролям и полномочиям</span>`
	
	// Проверяем извлеченный текст
	extractedText := extractTextFromHTML(blockContent)
	fmt.Printf("Извлеченный текст: '%s'\n", extractedText)
	
	normalizedExtracted := normalizeText(extractedText)
	normalizedTarget := normalizeText(target)
	
	fmt.Printf("Нормализованный извлеченный: '%s'\n", normalizedExtracted)
	fmt.Printf("Нормализованный целевой: '%s'\n", normalizedTarget)
	fmt.Printf("Содержит: %v\n", stringContainsSubstring(normalizedExtracted, normalizedTarget))
	
	// Проверяем размер блока
	reasonable := isBlockSizeReasonable(blockContent, target)
	fmt.Printf("Размер блока разумен: %v\n", reasonable)
	
	// Тестируем wrapWithinBlock
	fmt.Printf("\n=== wrapWithinBlock ===\n")
	wrapped1, found1, err1 := wrapWithinBlock(blockContent, target, "test1")
	fmt.Printf("wrapWithinBlock: found=%v, err=%v\n", found1, err1)
	if found1 {
		fmt.Printf("Результат: %s\n", wrapped1)
	}
	
	// Тестируем wrapInNestedSpans напрямую  
	fmt.Printf("\n=== wrapInNestedSpans ===\n")
	wrapped2, found2, err2 := wrapInNestedSpans(blockContent, target, "test2")
	fmt.Printf("wrapInNestedSpans: found=%v, err=%v\n", found2, err2)
	if found2 {
		fmt.Printf("Результат: %s\n", wrapped2)
	}
	
	// Тестируем findTextInSpanSequence
	fmt.Printf("\n=== findTextInSpanSequence ===\n")
	pos, len, err3 := findTextInSpanSequence(blockContent, normalizedTarget)
	fmt.Printf("findTextInSpanSequence: pos=%d, len=%d, err=%v\n", pos, len, err3)
	
	// Дополнительная отладка
	fmt.Printf("\n=== Отладка поиска span-ов ===\n")
	debugFindTextInSpanSequence(blockContent, normalizedTarget)
	
	if pos != -1 {
		candidate := blockContent[pos:pos+len]
		fmt.Printf("Кандидат: '%s'\n", candidate)
		
		isMatch := isNestedSpanMatch(candidate, target)
		fmt.Printf("isNestedSpanMatch: %v\n", isMatch)
	}
	
	// Полный тест
	fmt.Printf("\n=== Полный тест ===\n")
	result, found, err := WrapSubstringSmartHTML(html, target, "final-test")
	fmt.Printf("Результат: found=%v, err=%v\n", found, err)
	if found {
		fmt.Printf("HTML: %s\n", result)
	}
}

func debugFindTextInSpanSequence(htmlStr, normalizedTarget string) {
	fmt.Printf("HTML для поиска: %s\n", htmlStr)
	fmt.Printf("Целевой текст: %s\n", normalizedTarget)
	
	// Создаем регулярное выражение для поиска span-тегов
	spanPattern := regexp.MustCompile(`(?i)<span[^>]*>([^<]*)</span>`)
	matches := spanPattern.FindAllStringSubmatchIndex(htmlStr, -1)
	
	fmt.Printf("Найдено span матчей: %d\n", len(matches))
	
	if len(matches) == 0 {
		fmt.Printf("❌ Нет span тегов!\n")
		return
	}
	
	// Собираем текстовые фрагменты с их позициями
	var fragments []textFragment
	
	for i, match := range matches {
		if len(match) >= 4 {
			// match содержит: 0-1:полное совпадение, 2-3:содержимое span
			spanStart, spanEnd := match[0], match[1]
			contentStart, contentEnd := match[2], match[3]
			
			if contentStart != -1 && contentEnd != -1 {
				textContent := strings.TrimSpace(htmlStr[contentStart:contentEnd])
				if textContent != "" {
					fragments = append(fragments, textFragment{
						text:       textContent,
						htmlPos:    spanStart,
						htmlLen:    spanEnd - spanStart,
						normalized: normalizeText(textContent),
					})
					
					fmt.Printf("Span %d: '%s' -> нормализованный: '%s'\n", i, textContent, normalizeText(textContent))
				}
			}
		}
	}
	
	fmt.Printf("Всего фрагментов: %d\n", len(fragments))
	
	if len(fragments) == 0 {
		fmt.Printf("❌ Нет текстовых фрагментов!\n")
		return
	}
	
	// Пытаемся найти совпадение
	targetWords := strings.Fields(normalizedTarget)
	fmt.Printf("Слова целевого текста: %v\n", targetWords)
	
	// Тестируем скользящее окно
	for startIdx := 0; startIdx < len(fragments) && startIdx < 3; startIdx++ {
		fmt.Printf("\n--- Окно, начиная с фрагмента %d ---\n", startIdx)
		var combinedNormalized strings.Builder
		
		for endIdx := startIdx; endIdx < len(fragments) && endIdx < startIdx+10; endIdx++ {
			if combinedNormalized.Len() > 0 {
				combinedNormalized.WriteString(" ")
			}
			combinedNormalized.WriteString(fragments[endIdx].normalized)
			
			currentNormalized := combinedNormalized.String()
			
			exactMatch := strings.Contains(currentNormalized, normalizedTarget)
			similarity := calculateTextSimilarity(currentNormalized, normalizedTarget)
			
			fmt.Printf("  Окно[%d-%d]: '%s' -> точное: %v, схожесть: %.2f\n", 
				startIdx, endIdx, currentNormalized, exactMatch, similarity)
			
			if exactMatch || similarity > 0.8 {
				fmt.Printf("  ✅ Найдено совпадение!\n")
				return
			}
		}
	}
	
	fmt.Printf("❌ Совпадение не найдено в окнах\n")
}