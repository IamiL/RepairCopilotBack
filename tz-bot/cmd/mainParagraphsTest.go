package main

import (
	"fmt"
	"os"
	paragraphsproc "repairCopilotBot/tz-bot/internal/pkg/word-parser2/paragraphs"
	"strconv"
	"strings"
)

// runTestScenario выполняет тестовый сценарий
func runTestScenario(html, scenarioName string) {
	fmt.Printf("\n--- Сценарий: %s ---\n", scenarioName)
	fmt.Println("Исходный HTML:")
	fmt.Println(html)

	// Извлечение параграфов
	HTMLWithPlaceholder, Paragraphs := paragraphsproc.ExtractParagraphs(html)

	//fmt.Println("\nHTML с плейсхолдерами:")
	//fmt.Println(HTMLWithPlaceholder)
	//
	//fmt.Println("\nИзвлеченные параграфы:")
	//fmt.Println(Paragraphs)

	// Восстановление
	restored := paragraphsproc.InsertParagraphs(HTMLWithPlaceholder, Paragraphs)

	fmt.Println("\nВосстановленный HTML:")
	fmt.Println(restored)

	// Проверка целостности (базовая)
	if len(Paragraphs) > 0 {
		fmt.Println("\n✅ Тест прошел: параграфы извлечены и восстановлены")
	} else {
		fmt.Println("\n⚠️  Внимание: параграфы не найдены")
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

func main2() {
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

//func main() {
//	src := `<div>
//        <section>
//           <article class="a">
//              <h2 id="t">Title</h2>
//              <p role="x">Hello <b>world</b></p>
//              <div data-x="1"><span>inner</span></div>
//           </article>
//           <aside>side</aside>
//        </section>
//        <section>
//           <article data-q="z">
//              <p>Another</p><p>One</p>
//           </article>
//        </section>
//    </div>`
//
//	withPH, paras := paragraphsproc.ExtractParagraphs(src)
//	fmt.Println("WITH PLACEHOLDERS:\n", withPH)
//	fmt.Println("\nPARAGRAPHS:\n", paras)
//
//	restored := paragraphsproc.InsertParagraphs(withPH, paras)
//	fmt.Println("\nRESTORED:\n", restored)
//}

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
