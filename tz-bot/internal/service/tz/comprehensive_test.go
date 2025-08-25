package tzservice

import (
	"strings"
	"testing"
)

func TestComprehensiveRealWorldCases(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		target   string
		expected bool
	}{
		{
			name: "Case 1: Сложная MES система",
			html: `<p data-mapping-id="9a82dfe7-6317-4b91-94fd-f82e25522220" style="margin: 0pt; min-height: 1em; margin-top: 5pt; margin-bottom: 5pt; line-height: 1;"><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">   Требования к режимам функционирования системы . </span><span lang="en-US" style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">MES</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">-система должна поддерживать основной режим, в котором выполняет все свои основные функции</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">.</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;"> В основном режиме функционирования система должна обеспечивать: - работу пользовате</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">лей в рамках отведенной им роли.</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;"> Система также должна обеспечивать возможность проведения следующих работ: - программное обслуживание; - модернизацию системы; - устранение аварийных ситуаций в модулях и приложениях. В порядке развития должна быть предусмотрена возможность расширения и увеличения количества пользователей. Также необходимо предусмотреть возможность увеличения производительности системы, расширения емкости хранения данных.</span></p>`,
			target: `Требования к режимам функционирования системы . MES-система должна поддерживать основной режим, в котором выполняет все свои основные функции.`,
			expected: true,
		},
		{
			name: "Case 2: Текст не из начала блока",
			html: `<p data-mapping-id="b9cd88b1-23ab-44ce-af4b-89f6f8fd6aab" style="margin: 0pt; min-height: 1em; margin-top: 5pt; margin-bottom: 5pt; line-height: 1; text-align: justify;"><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>3</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>.1. </span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>Общая структура создаваемой системы</span></p>`,
			target: `В порядке развития должна быть предусмотрена возможность расширения и увеличения количества пользователей.`,
			expected: false, // Этот текст не в данном HTML блоке
		},
		{
			name: "Case 3: Текст с ведущим пробелом",
			html: `<p data-mapping-id="9a82dfe7-6317-4b91-94fd-f82e25522220" style="margin: 0pt; min-height: 1em; margin-top: 5pt; margin-bottom: 5pt; line-height: 1;"><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">   Требования к режимам функционирования системы . </span><span lang="en-US" style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">MES</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">-система должна поддерживать основной режим, в котором выполняет все свои основные функции</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">.</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;"> В основном режиме функционирования система должна обеспечивать: - работу пользовате</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">лей в рамках отведенной им роли.</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;"> Система также должна обеспечивать возможность проведения следующих работ: - программное обслуживание; - модернизацию системы; - устранение аварийных ситуаций в модулях и приложениях. В порядке развития должна быть предусмотрена возможность расширения и увеличения количества пользователей. Также необходимо предусмотреть возможность увеличения производительности системы, расширения емкости хранения данных.</span></p>`,
			target: ` Требования к режимам функционирования системы . MES-система должна поддерживать основной режим, в котором выполняет все свои основные функции.`,
			expected: true,
		},
		{
			name: "Case 4: TRUMF техническое описание",
			html: `<p data-mapping-id="bdde43dd-d5e1-4edf-9197-cb3de7896ab0" style="margin: 0pt; min-height: 1em; margin-bottom: 8.00pt; line-height: 1; text-align: justify;"><span lang="en-US" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" min-height: 12.00pt; font-size: 12pt;'>TRUMF</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" min-height: 12.00pt; font-size: 12pt;'>-00 - Труба стальная бесшовная </span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" min-height: 12.00pt; font-size: 12pt;'>горячедеформированнная</span></p>`,
			target: `TRUMF-00 - Труба стальная бесшовная горячедеформированнная`,
			expected: true,
		},
		{
			name: "Case 5: Производственное описание с дефисами",
			html: `<p data-mapping-id="fcb18c18-0f90-4cc5-b215-d4079c01581d" style="margin: 0pt; min-height: 1em; margin-bottom: 8.00pt; line-height: 1.08; text-indent: 35.40pt; text-align: justify;"><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">Назначение участка – приемка трубной заготовки на (склад 2110) АО «НТПЗ» из железнодорожного и автомобильного транспорта. Размещение трубной заготовки с номером ЕНС в стойках хранения и учет, порезка трубной заготовки на мерные длины, нагрев заготовки, передача нагретой заготовки на участок стана горячей прокатки (код материала -TRUMB-00). </span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">В цехе принято решение,  что полученная длинная заготовка - будет называться штанга, уже порезанная на </span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;">краты</span><span style="white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial; min-height: 12.00pt; font-size: 12pt;"> просто заготовка.</span></p>`,
			target: `В цехе принято решение, что полученная длинная заготовка - будет называться штанга, уже порезанная на краты просто заготовка.`,
			expected: true,
		},
		{
			name: "Case 6: Номер раздела с пробелами",
			html: `<p data-mapping-id="27e5ccae-cc60-482e-a7f4-aa79e0eefb19" style="margin: 0pt; min-height: 1em; margin-top: 5pt; margin-bottom: 5pt; line-height: 1; text-align: justify;"><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>3</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>.</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>2</span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'> </span><span lang="null" style='white-space: pre-wrap; overflow-wrap: break-word; font-family: Arial, " undefined: TimesNewRoman" font-weight: bold; min-height: 12.00pt; font-size: 12pt;'>Требование к ролям и полномочиям</span></p>`,
			target: `3.2 Требование к ролям и полномочиям`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found, err := WrapSubstringSmartHTML(tt.html, tt.target, "test-id-"+tt.name)
			
			if err != nil {
				t.Errorf("Неожиданная ошибка: %v", err)
				return
			}

			if found != tt.expected {
				t.Errorf("Ожидали found=%v, получили found=%v", tt.expected, found)
				
				// Дополнительная диагностика при неудаче
				if !found && tt.expected {
					t.Logf("HTML: %s", tt.html)
					t.Logf("Target: %s", tt.target)
					
					// Проверяем извлеченный текст
					extractedText := extractTextFromHTML(tt.html)
					t.Logf("Извлеченный текст: '%s'", extractedText)
					t.Logf("Нормализованный извлеченный: '%s'", normalizeText(extractedText))
					t.Logf("Нормализованный целевой: '%s'", normalizeText(tt.target))
					
					// Проверяем содержание
					contains := strings.Contains(normalizeText(extractedText), normalizeText(tt.target))
					t.Logf("Содержит целевой текст: %v", contains)
				}
				return
			}

			if found {
				// Проверяем, что результат содержит правильный error-id
				expectedErrorId := "test-id-" + tt.name
				if !strings.Contains(result, `error-id="`+expectedErrorId+`"`) {
					t.Errorf("Результат не содержит правильный error-id '%s'", expectedErrorId)
					t.Logf("Результат: %s", result)
				}
				
				t.Logf("✅ Успешно найден и обернут: %s", tt.name)
			} else {
				t.Logf("❌ Ожидаемо не найден: %s", tt.name)
			}
		})
	}
}

// Дополнительный тест для проверки нормализации
func TestTextNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Дефисы с пробелами",
			input:    "MES -система должна",
			expected: "mes-система должна",
		},
		{
			name:     "Точки с пробелами",
			input:    "системы . MES",
			expected: "системы.mes",
		},
		{
			name:     "Множественные пробелы",
			input:    "текст   с    множественными     пробелами",
			expected: "текст с множественными пробелами",
		},
		{
			name:     "Комплексный случай",
			input:    "   Требования к системы .  MES  -система  должна  ",
			expected: "требования к системы.mes-система должна",
		},
		{
			name:     "Пробелы перед точками",
			input:    "3 . 2 Требование",
			expected: "3.2 требование",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("Ожидали '%s', получили '%s'", tt.expected, result)
			}
		})
	}
}