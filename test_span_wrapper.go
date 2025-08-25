package main

import (
	"fmt"
	"log"

	tzservice "repairCopilotBot/tz-bot/internal/service/tz"
)

func main() {
	// Тестовый HTML из вашего примера
	testHTML := `<p style="margin: 0pt"><span style="white-space: pre-wrap">   Требования к режимам функционирования системы . </span><span lang="en-US" style="white-space: pre-wrap">MES</span><span style="white-space: pre-wrap">-система должна поддерживать основной режим, в котором выполняет все свои основные функции</span><span style="white-space: pre-wrap">.</span><span style="white-space: pre-wrap"> В основном режиме функционирования система должна обеспечивать: - работу пользователе</span><span style="white-space: pre-wrap">лей в рамках отведенной им роли.</span><span style="white-space: pre-wrap"> Система также должна обеспечивать возможность проведения следующих работ: - программное обслуживание; - модернизацию системы; - устранение аварийных ситуаций в модулях и приложениях. В порядке развития должна быть предусмотрена возможность расширения и увеличения количества пользователей. Также необходимо предусмотреть возможность увеличения производительности системы, расширения емкости хранения данных.</span></p>`
	
	// Искомая подстрока
	targetText := "Требования к режимам функционирования системы . MES-система должна поддерживать основной режим, в котором выполняет все свои основные функции."

	fmt.Println("=== Тестирование модернизированного алгоритма обёртывания ===")
	fmt.Printf("Исходный HTML (длина: %d):\n%s\n\n", len(testHTML), testHTML)
	fmt.Printf("Искомый текст (длина: %d):\n%s\n\n", len(targetText), targetText)

	// Вызываем функцию оборачивания
	result, found, err := tzservice.WrapSubstringSmartHTML(testHTML, targetText, "test-error-123")
	if err != nil {
		log.Fatalf("Ошибка: %v", err)
	}

	fmt.Printf("Результат: найдено = %v\n", found)
	if found {
		fmt.Printf("Обёрнутый HTML (длина: %d):\n%s\n", len(result), result)
		
		// Проверяем, что в результате есть наш span с error-id
		if contains := contains(result, `error-id="test-error-123"`); contains {
			fmt.Println("✅ Успешно: найден span с правильным error-id")
		} else {
			fmt.Println("❌ Ошибка: не найден span с error-id")
		}
	} else {
		fmt.Println("❌ Текст не найден!")
	}

	// Дополнительные тесты
	fmt.Println("\n=== Дополнительные тесты ===")
	
	// Тест 1: Простой случай
	simpleHTML := `<p>Это простой <span>тест</span> текста.</p>`
	simpleTarget := "простой тест текста"
	fmt.Printf("\nТест 1 - простой случай:\nHTML: %s\nИщем: %s\n", simpleHTML, simpleTarget)
	result1, found1, _ := tzservice.WrapSubstringSmartHTML(simpleHTML, simpleTarget, "test-1")
	fmt.Printf("Найдено: %v\nРезультат: %s\n", found1, result1)

	// Тест 2: Множественные span с пробелами
	multiSpanHTML := `<p><span>Это</span> <span>тест</span> <span>множественных</span> <span>span</span> <span>тегов</span></p>`
	multiSpanTarget := "тест множественных span тегов"
	fmt.Printf("\nТест 2 - множественные span:\nHTML: %s\nИщем: %s\n", multiSpanHTML, multiSpanTarget)
	result2, found2, _ := tzservice.WrapSubstringSmartHTML(multiSpanHTML, multiSpanTarget, "test-2")
	fmt.Printf("Найдено: %v\nРезультат: %s\n", found2, result2)

	// Тест 3: Частичное совпадение
	partialHTML := `<p><span>Первая</span> <span>часть</span> <span>текста</span> и <span>вторая</span></p>`
	partialTarget := "часть текста и"
	fmt.Printf("\nТест 3 - частичное совпадение:\nHTML: %s\nИщем: %s\n", partialHTML, partialTarget)
	result3, found3, _ := tzservice.WrapSubstringSmartHTML(partialHTML, partialTarget, "test-3")
	fmt.Printf("Найдено: %v\nРезультат: %s\n", found3, result3)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      stringContains(s, substr))))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}