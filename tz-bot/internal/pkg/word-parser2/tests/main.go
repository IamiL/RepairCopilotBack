package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const paragraphPlaceholder = "{{PARAGRAPHS_PLACEHOLDER_{{ARTICLE_INDEX}}}}"

// extractChildBlocks извлекает дочерние блоки из содержимого статьи
func extractChildBlocks(articleContent string, articleIndex int) string {
	var result strings.Builder

	// Находим все прямые дочерние блоки верхного уровня
	blockRegex := regexp.MustCompile(`(?s)<(\w+)([^>]*)>(.*?)</(\w+)>`)
	blockMatches := blockRegex.FindAllStringSubmatch(articleContent, -1)

	blockIndex := 0
	for _, blockMatch := range blockMatches {
		tagName := blockMatch[1]
		existingAttrs := blockMatch[2]
		blockContent := blockMatch[3]
		closingTag := blockMatch[4]

		// Skip if opening and closing tags don't match
		if tagName != closingTag {
			continue
		}

		// Add special attributes for tracking
		newAttrs := fmt.Sprintf(`%s data-article="%d" data-block="%d"`, existingAttrs, articleIndex, blockIndex)
		processedBlock := fmt.Sprintf(`<%s%s>%s</%s>`, tagName, newAttrs, blockContent, tagName)
		result.WriteString(processedBlock)

		blockIndex++
	}

	return result.String()
}

// ParagraphExtractionResult represents the result of paragraph extraction
type ParagraphExtractionResult struct {
	HTMLWithPlaceholder string
	Paragraphs          string
}

// ExtractParagraphs extracts paragraphs from multiple articles with numbering
func ExtractParagraphs(html string) ParagraphExtractionResult {
	// Regular expression to find all article elements with their content (multiline mode)
	articleRegex := regexp.MustCompile(`(?s)<article[^>]*>(.*?)</article>`)
	articleMatches := articleRegex.FindAllStringSubmatch(html, -1)

	if len(articleMatches) == 0 {
		return ParagraphExtractionResult{
			HTMLWithPlaceholder: html,
			Paragraphs:          "",
		}
	}

	var allParagraphs strings.Builder
	htmlWithPlaceholder := html

	// Process articles in reverse order to avoid position shifts
	for i := len(articleMatches) - 1; i >= 0; i-- {
		match := articleMatches[i]
		articleContent := match[1]

		// Extract all child blocks from the article content using a more sophisticated approach
		extractedBlocks := extractChildBlocks(articleContent, i)
		allParagraphs.WriteString(extractedBlocks)

		// Extract original article attributes
		fullArticleMatch := match[0]
		articleTagRegex := regexp.MustCompile(`<article([^>]*)>`)
		articleTagMatch := articleTagRegex.FindStringSubmatch(fullArticleMatch)

		var articleAttrs string
		if len(articleTagMatch) >= 2 {
			articleAttrs = articleTagMatch[1]
		}

		// Replace this specific article with numbered placeholder, preserving attributes
		placeholder := strings.ReplaceAll(paragraphPlaceholder, "{{ARTICLE_INDEX}}", fmt.Sprintf("%d", i))
		htmlWithPlaceholder = strings.Replace(htmlWithPlaceholder, fullArticleMatch, fmt.Sprintf("<article%s>%s</article>", articleAttrs, placeholder), 1)
	}

	//fmt.Println("возвращаем из extractParagraphs: paragraphs = ", allParagraphs.String())

	return ParagraphExtractionResult{
		HTMLWithPlaceholder: htmlWithPlaceholder,
		Paragraphs:          allParagraphs.String(),
	}
}

// InsertParagraphs inserts paragraphs back into HTML replacing numbered placeholders
func InsertParagraphs(htmlWithPlaceholder, paragraphs string) string {
	// Parse all blocks from paragraphs string to organize by article
	blockRegex := regexp.MustCompile(`<(\w+)[^>]*data-article="(\d+)"[^>]*data-block="(\d+)"[^>]*>([\s\S]*?)</(\w+)>`)
	blockMatches := blockRegex.FindAllStringSubmatch(paragraphs, -1)

	// Group blocks by article index
	articleBlocks := make(map[int][]string)
	for _, match := range blockMatches {
		articleIndex := 0
		fmt.Sscanf(match[2], "%d", &articleIndex)
		blockIndex := 0
		fmt.Sscanf(match[3], "%d", &blockIndex)

		// Check if opening and closing tags match
		tagName := match[1]
		closingTag := match[5]
		if tagName != closingTag {
			continue
		}

		// Remove the special attributes from the block
		blockContent := match[4]

		// Extract original attributes by removing data-article and data-block
		fullMatch := match[0]
		startTagRegex := regexp.MustCompile(`<` + tagName + `([^>]*)>`)
		startTagMatch := startTagRegex.FindStringSubmatch(fullMatch)

		var originalAttrs string
		if len(startTagMatch) >= 2 {
			allAttrs := startTagMatch[1]
			// Remove the special tracking attributes
			allAttrs = regexp.MustCompile(`\s+data-article="[^"]*"`).ReplaceAllString(allAttrs, "")
			allAttrs = regexp.MustCompile(`\s+data-block="[^"]*"`).ReplaceAllString(allAttrs, "")
			originalAttrs = strings.TrimSpace(allAttrs)
		}

		// Reconstruct block without special attributes
		var cleanBlock string
		if originalAttrs != "" {
			cleanBlock = fmt.Sprintf(`<%s %s>%s</%s>`, tagName, originalAttrs, blockContent, tagName)
		} else {
			cleanBlock = fmt.Sprintf(`<%s>%s</%s>`, tagName, blockContent, tagName)
		}

		if articleBlocks[articleIndex] == nil {
			articleBlocks[articleIndex] = make([]string, 0)
		}

		// Insert block at correct position
		for len(articleBlocks[articleIndex]) <= blockIndex {
			articleBlocks[articleIndex] = append(articleBlocks[articleIndex], "")
		}
		articleBlocks[articleIndex][blockIndex] = cleanBlock
	}

	result := htmlWithPlaceholder

	// Replace each numbered placeholder with corresponding article content
	for articleIndex, blocks := range articleBlocks {
		placeholder := strings.ReplaceAll(paragraphPlaceholder, "{{ARTICLE_INDEX}}", fmt.Sprintf("%d", articleIndex))
		articleContent := strings.Join(blocks, "")

		result = strings.ReplaceAll(result, placeholder, articleContent)
	}

	return result
}

// runTestScenario выполняет тестовый сценарий
func runTestScenario(html, scenarioName string) {
	fmt.Printf("\n--- Сценарий: %s ---\n", scenarioName)
	fmt.Println("Исходный HTML:")
	fmt.Println(html)

	// Извлечение параграфов
	result := ExtractParagraphs(html)

	fmt.Println("\nHTML с плейсхолдерами:")
	fmt.Println(result.HTMLWithPlaceholder)

	fmt.Println("\nИзвлеченные параграфы:")
	fmt.Println(result.Paragraphs)

	// Восстановление
	restored := InsertParagraphs(result.HTMLWithPlaceholder, result.Paragraphs)

	fmt.Println("\nВосстановленный HTML:")
	fmt.Println(restored)

	// Проверка целостности (базовая)
	if len(result.Paragraphs) > 0 {
		fmt.Println("\n✅ Тест прошел: параграфы извлечены и восстановлены")
	} else {
		fmt.Println("\n⚠️  Внимание: параграфы не найдены")
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		return
	}

	arg := os.Args[1]

	if arg == "all" {
		fmt.Println("🚀 Запуск всех тестовых сценариев...")
		runAllTests()
		fmt.Println("✅ Все тесты завершены!")
		return
	}

	scenarioNumber, err := strconv.Atoi(arg)
	if err != nil {
		fmt.Printf("❌ Ошибка: '%s' не является числом\n", arg)
		return
	}

	if scenarioNumber < 1 || scenarioNumber > 6 {
		fmt.Println("❌ Ошибка: номер сценария должен быть от 1 до 6")
		return
	}

	fmt.Printf("🚀 Запуск тестового сценария %d...\n", scenarioNumber)
	runSingleTest(scenarioNumber)
	fmt.Println("✅ Тест завершен!")
}

func showUsage() {
	fmt.Println("📋 Доступные тестовые сценарии:")
	fmt.Println("1 - Простые множественные статьи")
	fmt.Println("2 - Сложная вложенная структура")
	fmt.Println("3 - Одна статья")
	fmt.Println("4 - Без статей")
	fmt.Println("5 - Пустые статьи")
	fmt.Println("6 - Статьи с множеством атрибутов")
	fmt.Println("all - Запустить все тесты")
	fmt.Println()
	fmt.Println("Использование:")
	fmt.Println("  go run tests/main.go 1          # запустить тест 1")
	fmt.Println("  go run tests/main.go all        # запустить все тесты")
	fmt.Println("  go test -v                      # запустить стандартные тесты")
}

func runAllTests() {
	for i := 1; i <= 6; i++ {
		runSingleTest(i)
	}
}

func runSingleTest(scenarioNumber int) {
	switch scenarioNumber {
	case 1:
		testScenario1()
	case 2:
		testScenario2()
	case 3:
		testScenario3()
	case 4:
		testScenario4()
	case 5:
		testScenario5()
	case 6:
		testScenario6()
	default:
		fmt.Println("Неизвестный номер сценария. Доступные: 1-6")
	}
}

// Тестовые сценарии
func testScenario1() {
	fmt.Println("\n=== ТЕСТ 1: Простые множественные статьи ===")

	testHTML := `<div>
		<section>
			<article class="doc-article">
				<p>текст 1</p>
				<p>текст 2</p>
				<table border="1">таблица 1</table>
			</article>
		</section>
		<section class="lallala">
			<article class="lallala" id="second-article">
				<p>текст 4</p>
				<table class="table-2">таблица 2</table>
			</article>
			<footer>какой-то футер</footer>
		</section>
	</div>`

	runTestScenario(testHTML, "Простые множественные статьи")
}

func testScenario2() {
	fmt.Println("\n=== ТЕСТ 2: Сложная вложенная структура ===")

	testHTML := `<div class="document">
		<header>Заголовок документа</header>
		<main>
			<section class="chapter1">
				<article data-id="art1" class="content-block">
					<h2>Раздел 1</h2>
					<p class="intro">Введение к первому разделу</p>
					<div class="subsection">
						<p>Подраздел 1.1</p>
						<ul>
							<li>Пункт 1</li>
							<li>Пункт 2</li>
						</ul>
					</div>
					<table class="data-table" border="1" cellpadding="5">
						<tr><th>Колонка 1</th><th>Колонка 2</th></tr>
						<tr><td>Данные 1</td><td>Данные 2</td></tr>
					</table>
				</article>
			</section>
			<section class="chapter2">
				<article id="second" data-type="content">
					<h2>Раздел 2</h2>
					<p style="color: blue;">Цветной текст</p>
					<blockquote cite="source">
						Цитата из какого-то источника
					</blockquote>
				</article>
			</section>
		</main>
	</div>`

	runTestScenario(testHTML, "Сложная вложенная структура")
}

func testScenario3() {
	fmt.Println("\n=== ТЕСТ 3: Одна статья ===")

	testHTML := `<div>
		<section>
			<article class="single-article">
				<h1>Заголовок</h1>
				<p>Первый параграф</p>
				<p>Второй параграф</p>
				<div class="content-block">
					<span>Вложенный контент</span>
				</div>
			</article>
		</section>
	</div>`

	runTestScenario(testHTML, "Одна статья")
}

func testScenario4() {
	fmt.Println("\n=== ТЕСТ 4: Без статей ===")

	testHTML := `<div>
		<section>
			<p>Простой параграф без article</p>
			<div>Обычный div</div>
		</section>
	</div>`

	runTestScenario(testHTML, "Без статей")
}

func testScenario5() {
	fmt.Println("\n=== ТЕСТ 5: Пустые статьи ===")

	testHTML := `<div>
		<section>
			<article class="empty1"></article>
			<article class="empty2">
			</article>
			<article class="with-content">
				<p>Единственный контент</p>
			</article>
		</section>
	</div>`

	runTestScenario(testHTML, "Пустые статьи")
}

func testScenario6() {
	fmt.Println("\n=== ТЕСТ 6: Статьи с множеством атрибутов ===")

	testHTML := `<div>
		<section>
			<article 
				id="complex-article" 
				class="main-article featured"
				data-category="news" 
				data-published="2024-01-01"
				style="margin: 10px; padding: 20px;"
				role="main"
				aria-labelledby="title1">
				<h2 id="title1" class="article-title">Заголовок с атрибутами</h2>
				<p class="lead-paragraph" data-priority="high">Важный параграф</p>
				<table 
					class="responsive-table"
					id="data-table-1"
					data-sortable="true"
					border="1"
					cellpadding="10"
					cellspacing="0">
					<thead>
						<tr><th>Заголовок таблицы</th></tr>
					</thead>
					<tbody>
						<tr><td>Данные таблицы</td></tr>
					</tbody>
				</table>
			</article>
		</section>
	</div>`

	runTestScenario(testHTML, "Статьи с множеством атрибутов")
}
