package llmClient

import (
	"strings"
	"testing"
)

func TestMarkdownToHTML(t *testing.T) {
	// Test case with the example provided by user
	input := `1️⃣ **Есть ли информация по запросу?** ДА

2️⃣ **Рекомендации по вопросу:**
- Для замены двигателя, если невозможно установить другой двигатель, можно использовать переходную плиту. Это решение позволяет адаптировать крепления под новый двигатель и избежать сложностей с монтажом (идея: '_2020_06_2816_A_УС_АТЦ_', статус: '6. Внедрение').
- Если двигатель часто ломается из-за вибрации или потери соединительных элементов, рекомендуется установить зубчатую муфту. Это решение помогает устранить вибрацию и предотвратить повреждение обмотки двигателя (идея: '_2020_011_6595_A_УС_ДЦ_', статус: 'Реализовано').
- Для защиты двигателя от повреждений и продления срока его службы, можно накрыть его куском резины. Это простое решение может предотвратить попадание посторонних предметов и влаги (идея: '_2021_09_10630_Полезная идея_УС_ЦРМО_', статус: 'Реализовано').
- Если двигатель часто выходит из строя из-за обрыва фаз, установка реле обрыва фаз может быть эффективным решением. Оно автоматически отключает питание двигателя при обнаружении проблемы, предотвращая дальнейшие повреждения (идея: '_2019_010_637_B_УС_КХП_', статус: '8. Премия выплачена').

Эти рекомендации основаны на опыте решения схожих проблем и могут быть полезны для устранения неисправности двигателя.

3️⃣ **Ответ основан на:**
- '_2020_06_2816_A_УС_АТЦ_'
- '_2020_011_6595_A_УС_ДЦ_'
- '_2021_09_10630_Полезная идея_УС_ЦРМО_'
- '_2019_010_637_B_УС_КХП_'`

	result := MarkdownToHTML(input)

	// Check that emojis are preserved
	if !strings.Contains(result, "1️⃣") {
		t.Error("Emoji should be preserved")
	}

	// Check that bold text is converted
	if !strings.Contains(result, "<strong>Есть ли информация по запросу?</strong>") {
		t.Error("Bold text should be converted to <strong> tags")
	}

	// Check that list items are converted
	if !strings.Contains(result, "<ul>") {
		t.Error("Lists should be wrapped in <ul> tags")
	}

	if !strings.Contains(result, "<li>") {
		t.Error("List items should be wrapped in <li> tags")
	}

	// Print the result for visual inspection
	t.Logf("Result:\n%s", result)
}

func TestMarkdownToHTML_BoldText(t *testing.T) {
	input := "This is **bold** text"
	expected := "This is <strong>bold</strong> text"
	result := MarkdownToHTML(input)

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

func TestMarkdownToHTML_Lists(t *testing.T) {
	input := `List example:
- Item 1
- Item 2
- Item 3`

	result := MarkdownToHTML(input)

	if !strings.Contains(result, "<ul>") {
		t.Error("Should contain <ul> tag")
	}

	if !strings.Contains(result, "<li>Item 1</li>") {
		t.Error("Should contain list items wrapped in <li> tags")
	}

	if !strings.Contains(result, "</ul>") {
		t.Error("Should close <ul> tag")
	}
}

func TestMarkdownToHTML_Mixed(t *testing.T) {
	input := `**Title**
- Item with **bold**
- Another item

Normal text`

	result := MarkdownToHTML(input)

	// Should have bold converted
	if !strings.Contains(result, "<strong>") {
		t.Error("Should convert bold text")
	}

	// Should have lists
	if !strings.Contains(result, "<ul>") || !strings.Contains(result, "</ul>") {
		t.Error("Should have list tags")
	}

	t.Logf("Result:\n%s", result)
}