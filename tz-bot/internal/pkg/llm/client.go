package tz_llm_client

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Url string `yaml:"url"`
}

type Client struct {
	url string // URL API для отправки файла (замените на реальный URL) apiURL := "https://your-api-endpoint.com/upload"
}

func New(url string) *Client {
	return &Client{url: url}
}

// Request структура для отправки запроса
type Request struct {
	Markdown string `json:"markdown"`
	Model    string `json:"model"`
	//Codes []string `json:"codes"`
}

// Retrieval структура для извлеченного текста
type Retrieval struct {
	Text      string `json:"text"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// Process структура для процесса анализа
type Process struct {
	Retrieval    []Retrieval `json:"retrieval"`
	Analysis     string      `json:"analysis"`
	Critique     string      `json:"critique"`
	Verification string      `json:"verification"`
}

// Instance структура для экземпляра ошибки
type Instance struct {
	ErrType      string `json:"err_type"`
	Snippet      string `json:"snippet"`
	LineStart    *int   `json:"line_start"`
	LineEnd      *int   `json:"line_end"`
	SuggestedFix string `json:"suggested_fix"`
	Rationale    string `json:"rationale"`
}

// ReportError структура для ошибки в отчете
type ReportError struct {
	Code      string     `json:"code"`
	Process   Process    `json:"process"`
	Verdict   string     `json:"verdict"`
	Instances []Instance `json:"instances"`
}

// Report структура для отчета группы
type Report struct {
	GroupId          string        `json:"group_id"`
	PreliminaryNotes string        `json:"preliminary_notes"`
	Errors           []ReportError `json:"errors"`
	OverallCritique  *string       `json:"overall_critique"`
}

// Tokens структура для информации о токенах
type Tokens struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
	Total      int `json:"total"`
}

// SuccessResponse структура для успешного ответа (200)
type SuccessResponse struct {
	Reports []Report `json:"reports"`
	Tokens  Tokens   `json:"tokens"`
}

// ValidationError структура для ошибки валидации (422)
type ValidationError struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

// ErrorResponse структура для ответа с ошибкой валидации
type ErrorResponse struct {
	Detail []ValidationError `json:"detail"`
}

// APIResponse общая структура ответа
type APIResponse struct {
	Success *SuccessResponse
	Error   *ErrorResponse
	Status  int
}

// MakeHTTPRequest выполняет HTTP запрос к API (MOCK - возвращает данные из файла)
func (c *Client) MakeHTTPRequest(req Request) (*APIResponse, error) {
	// ВРЕМЕННЫЙ МОК - читаем JSON из файла вместо HTTP запроса
	fmt.Println("MOCK: Используется тестовый ответ из response_example.json")

	mockData, err := json.Marshal(req) // Просто для логирования
	if err == nil {
		fmt.Printf("MOCK: Запрос был бы отправлен: %s\n", string(mockData))
	}

	// Читаем мок-ответ из файла
	mockFilePath := "tz-bot/internal/pkg/llm/response_example.json"
	mockBody, err := os.ReadFile(mockFilePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения мок-файла %s: %w", mockFilePath, err)
	}

	result := &APIResponse{
		Status: 200, // Всегда возвращаем успешный статус для мока
	}

	// Парсим мок-ответ как успешный
	var successResp SuccessResponse
	if err := json.Unmarshal(mockBody, &successResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга мок-ответа: %w", err)
	}

	fmt.Printf("MOCK: количество отчетов: %d\n", len(successResp.Reports))
	totalErrors := 0
	for _, report := range successResp.Reports {
		for _, err := range report.Errors {
			if err.Verdict == "error_present" {
				totalErrors++
			}
		}
	}
	fmt.Printf("MOCK: общее количество найденных ошибок: %d\n", totalErrors)
	result.Success = &successResp

	return result, nil
}

// Пример использования
func (c *Client) Analyze(doc string) (*SuccessResponse, error) {
	// Создаем запрос
	req := Request{
		Markdown: doc,
		//Model: "yandexgpt/latest",
		//Codes: []string{"code1", "code2", "code3"},
	}

	// Выполняем запрос
	response, err := c.MakeHTTPRequest(req)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return nil, err
	}

	// Обрабатываем ответ
	switch response.Status {
	case 200:
		//fmt.Println("Успешный ответ:")
		//fmt.Printf("Количество ошибок: %d\n", len(response.Success.Errors))
		//fmt.Printf("Токены: prompt=%d, completion=%d, total=%d\n",
		//	response.Success.Tokens.Prompt,
		//	response.Success.Tokens.Completion,
		//	response.Success.Tokens.Total)

		return response.Success, nil
		// Выводим информацию об ошибках
		//for i, err := range response.Success.Errors {
		//	fmt.Printf("Ошибка %d: %s (%s)\n", i+1.txt, err.Title, err.Code)
		//	for j, finding := range err.Findings {
		//		fmt.Printf("  Finding %d: %s\n", j+1.txt, finding.Advice)
		//	}
		//}

	case 422:
		fmt.Println("Ошибка валидации:")
		for i, detail := range response.Error.Detail {
			fmt.Printf("Ошибка %d: %s (тип: %s, поле: %v)\n",
				i+1, detail.Msg, detail.Type, detail.Loc)
		}
	}

	return nil, nil
}
