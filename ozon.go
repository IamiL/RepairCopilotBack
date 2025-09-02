// main.go
// Go 1.22+ — 5 реализаций строго про параллельность.
// Задача: на вход — слайс URL; на выход — для каждого URL только true/false (доступен ли).
// Никаких ретраев, бэк-оффов, метрик — только разные паттерны конкуренции.
// Запуск:
//   go run .            # все подходы
//   go run . -mode 3    # только конкретный подход
// Полезные флаги:
//   -c   (параллельность для 3/4/5)
//   -w   (число воркеров для 4)

package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

var urls = []string{
	"http://ozon.ru",
	"https://ozon.ru",
	"http://google.com",
	"http://somesite.com",
	"http://non-existent.domain.tld",
	"https://ya.ru",
	"http://ya.ru",
	"http://ёёёё",
}

// ===== модель результата =====

type BoolResult struct {
	URL string
	OK  bool
}

// Один переиспользуемый http-клиент с таймаутом (простая и правильная практика)
var httpClient = &http.Client{Timeout: 5 * time.Second}

// Проверка доступности: простой GET, код < 400 — OK.
func check(url string) bool {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 400
}

func printReport(title string, results []BoolResult) {
	fmt.Println("\n=== " + title + " ===")
	for _, r := range results {
		fmt.Printf("%-35s => %t\n", r.URL, r.OK)
	}
}

// ===== 1) Последовательно (baseline) =====
func seq(urls []string) []BoolResult {
	out := make([]BoolResult, 0, len(urls))
	for _, u := range urls {
		out = append(out, BoolResult{u, check(u)})
	}
	return out
}

// ===== 2) Fan-out/Fan-in: горутина на URL + WaitGroup + results-канал =====
func fanOut(urls []string) []BoolResult {
	resCh := make(chan BoolResult)
	var wg sync.WaitGroup
	wg.Add(len(urls))
	for _, u := range urls {
		u := u // локальная копия
		go func() {
			defer wg.Done()
			resCh <- BoolResult{u, check(u)}
		}()
	}
	// закрываем канал, когда все горутины завершатся
	go func() { wg.Wait(); close(resCh) }()

	out := make([]BoolResult, 0, len(urls))
	for r := range resCh {
		out = append(out, r)
	}
	return out
}

// ===== 3) Ограничение параллельности семафором (буферизованный канал) =====
// Горутина на каждый URL остаётся, но одновременно выполняется не более N.
func limited(urls []string, maxConcurrent int) []BoolResult {
	sem := make(chan struct{}, maxConcurrent)
	resCh := make(chan BoolResult)
	var wg sync.WaitGroup
	wg.Add(len(urls))
	for _, u := range urls {
		u := u
		go func() {
			defer wg.Done()
			sem <- struct{}{} // acquire
			ok := check(u)
			<-sem // release
			resCh <- BoolResult{u, ok}
		}()
	}
	go func() { wg.Wait(); close(resCh) }()
	out := make([]BoolResult, 0, len(urls))
	for r := range resCh {
		out = append(out, r)
	}
	return out
}

// ===== 4) Worker Pool (фиксированное число воркеров) =====
// N воркеров читают задания из jobs и пишут результат в results.
func workerPool(urls []string, workers int) []BoolResult {
	jobs := make(chan string)
	results := make(chan BoolResult)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for u := range jobs {
			results <- BoolResult{u, check(u)}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	go func() { // producer
		for _, u := range urls {
			jobs <- u
		}
		close(jobs)
	}()

	go func() { // closer для results
		wg.Wait()
		close(results)
	}()

	out := make([]BoolResult, 0, len(urls))
	for r := range results {
		out = append(out, r)
	}
	return out
}

// ===== 5) errgroup + SetLimit (каноничный групповой запуск с ограничением) =====
// Преимущество: аккуратный контроль числа параллельных задач без явных семафоров/воркеров.
// Дополнительно показываем «правильный» захват индекса цикла, чтобы писать в нужное место слайса.
func withErrGroup(urls []string, maxConcurrent int) []BoolResult {
	g := new(errgroup.Group)
	g.SetLimit(maxConcurrent)

	out := make([]BoolResult, len(urls)) // сохраняем порядок как во входном списке
	for i, u := range urls {
		i, u := i, u // локальные копии для замыкания
		g.Go(func() error {
			out[i] = BoolResult{u, check(u)}
			return nil // ошибок как таковых мы не эскалируем — задача bool-овая
		})
	}
	_ = g.Wait() // ждать завершения всех
	return out
}

func main() {
	mode := flag.Int("mode", 0, "Какой подход выполнить (0 — все, 1..5 — конкретный)")
	concurrency := flag.Int("c", 4, "Ограничение параллельности для 3/5")
	workers := flag.Int("w", 4, "Число воркеров для 4 (worker pool)")
	flag.Parse()

	run := func(num int, name string, f func() []BoolResult) {
		if *mode == 0 || *mode == num {
			printReport(fmt.Sprintf("%d) %s", num, name), f())
		}
	}

	run(1, "Последовательно", func() []BoolResult { return seq(urls) })
	run(2, "Fan-out/Fan-in (WG + chan)", func() []BoolResult { return fanOut(urls) })
	run(3, "Семафор (ограниченная параллельность)", func() []BoolResult { return limited(urls, *concurrency) })
	run(4, "Worker Pool", func() []BoolResult { return workerPool(urls, *workers) })
	run(5, "errgroup + SetLimit", func() []BoolResult { return withErrGroup(urls, *concurrency) })
}
