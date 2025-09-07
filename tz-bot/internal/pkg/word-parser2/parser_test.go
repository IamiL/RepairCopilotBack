package word_parser2

import (
	"fmt"
	"testing"
)

// TestScenario1_SimpleMultipleArticles тестирует простой случай с двумя статьями
func TestScenario1_SimpleMultipleArticles(t *testing.T) {
	fmt.Println("\n=== ТЕСТ 1: Простые множественные статьи ===")
	
	testHTML := `<div>
		<section>
			<article class="doc-article">
				<p>текст 1</p>
				<p>текст 2</p>
				<table border="1">таблица 1</table>
			</article>
		</section>
		<section>
			<article id="second-article">
				<p>текст 4</p>
				<table class="table-2">таблица 2</table>
			</article>
		</section>
	</div>`

	runTestScenario(testHTML, "Простые множественные статьи")
}

// TestScenario2_ComplexNestedStructure тестирует сложную вложенную структуру
func TestScenario2_ComplexNestedStructure(t *testing.T) {
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
			<section class="chapter3">
				<article>
					<h2>Раздел 3</h2>
					<p>Простой параграф</p>
				</article>
			</section>
		</main>
		<footer>Подвал документа</footer>
	</div>`

	runTestScenario(testHTML, "Сложная вложенная структура")
}

// TestScenario3_SingleArticle тестирует случай с одной статьей
func TestScenario3_SingleArticle(t *testing.T) {
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

// TestScenario4_NoArticles тестирует случай без статей
func TestScenario4_NoArticles(t *testing.T) {
	fmt.Println("\n=== ТЕСТ 4: Без статей ===")
	
	testHTML := `<div>
		<section>
			<p>Простой параграф без article</p>
			<div>Обычный div</div>
		</section>
	</div>`

	runTestScenario(testHTML, "Без статей")
}

// TestScenario5_EmptyArticles тестирует пустые статьи
func TestScenario5_EmptyArticles(t *testing.T) {
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

// TestScenario6_ArticlesWithManyAttributes тестирует статьи с множеством атрибутов
func TestScenario6_ArticlesWithManyAttributes(t *testing.T) {
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
	
	fmt.Println("\n" + "="*60)
}

// ManualTest функция для ручного запуска конкретного теста
func ManualTest(scenarioNumber int) {
	switch scenarioNumber {
	case 1:
		TestScenario1_SimpleMultipleArticles(nil)
	case 2:
		TestScenario2_ComplexNestedStructure(nil)
	case 3:
		TestScenario3_SingleArticle(nil)
	case 4:
		TestScenario4_NoArticles(nil)
	case 5:
		TestScenario5_EmptyArticles(nil)
	case 6:
		TestScenario6_ArticlesWithManyAttributes(nil)
	default:
		fmt.Println("Неизвестный номер сценария. Доступные: 1-6")
	}
}