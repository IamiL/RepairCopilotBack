package main

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

httpClient := &http

func fetch1