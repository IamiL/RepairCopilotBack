package tzservice

import (
	"strings"
	"testing"
)

func TestWrapInComplexSpanStructure(t *testing.T) {
	// Тестовый HTML из примера пользователя
	testHTML := `<p style="margin: 0pt"><span style="white-space: pre-wrap">   Требования к режимам функционирования системы . </span><span lang="en-US" style="white-space: pre-wrap">MES</span><span style="white-space: pre-wrap">-система должна поддерживать основной режим, в котором выполняет все свои основные функции</span><span style="white-space: pre-wrap">.</span><span style="white-space: pre-wrap"> В основном режиме функционирования система должна обеспечивать: - работу пользовате</span><span style="white-space: pre-wrap">лей в рамках отведенной им роли.</span><span style="white-space: pre-wrap"> Система также должна обеспечивать возможность проведения следующих работ: - программное обслуживание; - модернизацию системы; - устранение аварийных ситуаций в модулях и приложениях. В порядке развития должна быть предусмотрена возможность расширения и увеличения количества пользователей. Также необходимо предусмотреть возможность увеличения производительности системы, расширения емкости хранения данных.</span></p>`

	// Искомая подстрока
	targetText := "Требования к режимам функционирования системы . MES-система должна поддерживать основной режим, в котором выполняет все свои основные функции"

	t.Logf("Тестируем поиск текста в сложной span-структуре")
	t.Logf("HTML длина: %d", len(testHTML))
	t.Logf("Искомый текст длина: %d", len(targetText))

	result, found, err := WrapSubstringSmartHTML(testHTML, targetText, "test-error-123")

	if err != nil {
		t.Fatalf("Неожиданная ошибка: %v", err)
	}

	if !found {
		t.Errorf("Текст не найден, а должен был быть найден")
		return
	}

	// Проверяем, что результат содержит наш span с правильным error-id
	if !strings.Contains(result, `error-id="test-error-123"`) {
		t.Errorf("Результат не содержит span с правильным error-id")
		t.Logf("Результат: %s", result)
		return
	}

	t.Logf("✅ Успешно найден и обернут текст в сложной span-структуре")
	t.Logf("Результат: %s", result)
}

func TestWrapInMultipleSpans(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		target   string
		expected bool
	}{
		{
			name:     "Простые множественные span",
			html:     `<p><span>Это</span> <span>тест</span> <span>множественных</span> <span>span</span></p>`,
			target:   "тест множественных span",
			expected: true,
		},
		{
			name:     "Span с разделителями",
			html:     `<p><span>Первый</span>, <span>второй</span> и <span>третий</span></p>`,
			target:   "второй и третий",
			expected: true,
		},
		{
			name:     "Nested span структура",
			html:     `<p><span>Начало <span>вложенного</span> текста</span> <span>конец</span></p>`,
			target:   "вложенного текста конец",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found, err := WrapSubstringSmartHTML(tt.html, tt.target, "test-id")

			if err != nil {
				t.Fatalf("Неожиданная ошибка: %v", err)
			}

			if found != tt.expected {
				t.Errorf("Ожидали found=%v, получили found=%v", tt.expected, found)
				if found {
					t.Logf("Результат: %s", result)
				}
				return
			}

			if found {
				if !strings.Contains(result, `error-id="test-id"`) {
					t.Errorf("Результат не содержит span с правильным error-id")
					t.Logf("Результат: %s", result)
				}
			}
		})
	}
}

func TestTextFragmentProcessing(t *testing.T) {
	// Тестируем внутренние функции обработки текстовых фрагментов
	testHTML := `<span>Первый</span> <span>второй</span> <span>третий</span>`
	targetText := "второй третий"

	position, length, err := findTextInSpanSequence(testHTML, normalizeText(targetText))
	if err != nil {
		t.Fatalf("Ошибка в findTextInSpanSequence: %v", err)
	}

	if position == -1 {
		t.Errorf("Не удалось найти последовательность span-ов")
		return
	}

	candidateHTML := testHTML[position : position+length]
	t.Logf("Найденный HTML фрагмент: %s", candidateHTML)

	// Проверяем, что найденный фрагмент соответствует нашим ожиданиям
	extractedText := extractTextFromHTML(candidateHTML)
	normalizedExtracted := normalizeText(extractedText)
	normalizedTarget := normalizeText(targetText)

	if !strings.Contains(normalizedExtracted, normalizedTarget) {
		t.Errorf("Извлеченный текст '%s' не содержит целевой текст '%s'", normalizedExtracted, normalizedTarget)
	}
}

func TestCalculateTextSimilarity(t *testing.T) {
	tests := []struct {
		text1    string
		text2    string
		expected float64
		minSim   float64
	}{
		{"hello world test", "hello world test", 1.0, 1.0},
		{"hello world", "hello test", 0.5, 0.4},
		{"completely different", "absolutely various", 0.0, 0.0},
		{"система должна поддерживать", "система должна обеспечивать", 0.5, 0.4},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			similarity := calculateTextSimilarity(tt.text1, tt.text2)

			if tt.expected > 0 && similarity != tt.expected {
				// Для точных совпадений проверяем точность
				if tt.expected == 1.0 && similarity != 1.0 {
					t.Errorf("Ожидали точное совпадение (1.0), получили %f", similarity)
				}
			}

			if similarity < tt.minSim {
				t.Errorf("Схожесть %f меньше минимального порога %f для '%s' vs '%s'",
					similarity, tt.minSim, tt.text1, tt.text2)
			}
		})
	}
}

func TestIsNestedSpanMatch(t *testing.T) {
	tests := []struct {
		name          string
		candidateHTML string
		originalText  string
		shouldMatch   bool
	}{
		{
			name:          "Точное соответствие",
			candidateHTML: `<span>тест текста</span>`,
			originalText:  "тест текста",
			shouldMatch:   true,
		},
		{
			name:          "Множественные span",
			candidateHTML: `<span>тест</span> <span>текста</span>`,
			originalText:  "тест текста",
			shouldMatch:   true,
		},
		{
			name:          "Слишком длинный HTML",
			candidateHTML: strings.Repeat("<span>test</span>", 100),
			originalText:  "test",
			shouldMatch:   false,
		},
		{
			name:          "Низкая схожесть",
			candidateHTML: `<span>совершенно другой</span> <span>текст</span>`,
			originalText:  "тест системы",
			shouldMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNestedSpanMatch(tt.candidateHTML, tt.originalText)
			if result != tt.shouldMatch {
				t.Errorf("Ожидали %v, получили %v для HTML: %s, текст: %s",
					tt.shouldMatch, result, tt.candidateHTML, tt.originalText)
			}
		})
	}
}
