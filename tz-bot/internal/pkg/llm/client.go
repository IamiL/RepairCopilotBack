package tz_llm_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	Url   string `yaml:"url"`
	Model string `yaml:"model"`
}

type Client struct {
	url   string // URL API для отправки файла (замените на реальный URL) apiURL := "https://your-api-endpoint.com/upload"
	model string
}

func New(url string, model string) *Client {
	return &Client{url: url, model: model}
}

// Request структура для отправки запроса
type Request struct {
	Markdown string `json:"markdown"`
	Model    string `json:"model"`
	//Codes []string `json:"codes"`
}

// Структуры для десериализации JSON
type ReportFile struct {
	Reports []GroupReport `json:"reports"`
}

type GroupReport struct {
	GroupID *string        `json:"group_id"`
	Errors  *[]ErrorReport `json:"errors"`
}

type ErrorReport struct {
	Code      *string     `json:"code"`
	Instances *[]instance `json:"instances"`
	Process   *Process    `json:"process"`
}

type Process struct {
	Analysis     *string `json:"analysis"`
	Critique     *string `json:"critique"`
	Verification *string `json:"verification"`
}

type instance struct {
	ErrType      *string `json:"err_type"`
	Snippet      *string `json:"snippet"`
	LineStart    *int    `json:"line_start"`
	LineEnd      *int    `json:"line_end"`
	SuggestedFix *string `json:"suggested_fix"`
	Rationale    *string `json:"rationale"`
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
	Success *ReportFile
	Error   *ErrorResponse
	Status  int
}

// MakeHTTPRequest выполняет реальный HTTP запрос к LLM API
func (c *Client) MakeHTTPRequest(req Request) (*APIResponse, error) {
	// Устанавливаем модель из конфигурации клиента
	req.Model = c.model
	
	// Сериализуем запрос в JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	fmt.Printf("Отправляем запрос к LLM API: %s\n", c.url)

	// Создаем HTTP запрос
	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	// Устанавливаем заголовки
	httpReq.Header.Set("Content-Type", "application/json")

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 120 * time.Second, // Увеличенный таймаут для LLM запросов
	}

	// Выполняем запрос
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения тела ответа: %w", err)
	}

	result := &APIResponse{
		Status: resp.StatusCode,
	}

	// Обрабатываем ответ в зависимости от статус кода
	switch resp.StatusCode {
	case 200:
		// Успешный ответ - парсим как ReportFile
		var successResp ReportFile
		if err := json.Unmarshal(body, &successResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга успешного ответа: %w", err)
		}
		result.Success = &successResp

	case 422:
		// Ошибка валидации - парсим как ErrorResponse
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга ответа с ошибкой валидации: %w", err)
		}
		result.Error = &errorResp

	default:
		// Другие коды ответов
		return nil, fmt.Errorf("неожиданный статус код: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

	return result, nil
}

// Пример использования
func (c *Client) Analyze(doc string) (*ReportFile, error) {
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
