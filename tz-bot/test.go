package main

import (
	"fmt"
	"strings"
)

// TrimPipesAndSpaces удаляет вертикальные палки и пробелы с начала и конца строки
func TrimPipesAndSpaces(s string) string {
	return strings.Trim(s, "| ")
}

func main() {
	// Тестовые примеры
	testCases := []string{
		"| |   | какой-то текст",
		"| какой-то текст |",
		"|  | | какой-то текст |  | |",
		"   | обычный текст |   ",
		"||||  текст без пробелов  ||||",
		"просто текст",
		"| | |",
		"",
	}

	fmt.Println("Тестирование функции TrimPipesAndSpaces:")
	fmt.Println(strings.Repeat("-", 50))

	for i, test := range testCases {
		result := TrimPipesAndSpaces(test)
		fmt.Printf("Тест %d:\n", i+1)
		fmt.Printf("  Входная строка: %q\n", test)
		fmt.Printf("  Результат:      %q\n", result)
		fmt.Println()
	}
}
