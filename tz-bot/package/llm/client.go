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
	HTML string `json:"html"`
	//Codes []string `json:"codes"`
}

// Finding структура для отдельного finding
type Finding struct {
	Paragraph string `json:"paragraph"`
	Quote     string `json:"quote"`
	Advice    string `json:"advice"`
}

// Error структура для ошибки в ответе
type Error struct {
	Code     string    `json:"code"`
	Title    string    `json:"title"`
	Kind     string    `json:"kind"`
	Findings []Finding `json:"findings"`
}

// Tokens структура для информации о токенах
type Tokens struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
	Total      int `json:"total"`
}

// SuccessResponse структура для успешного ответа (200)
type SuccessResponse struct {
	Errors []Error `json:"errors"`
	Tokens Tokens  `json:"tokens"`
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

// MakeHTTPRequest выполняет HTTP запрос к API
func (c *Client) MakeHTTPRequest(req Request) (*APIResponse, error) {
	// Сериализуем запрос в JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Создаем HTTP запрос
	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Устанавливаем заголовки
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Выполняем запрос
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	result := &APIResponse{
		Status: resp.StatusCode,
	}

	// Обрабатываем ответ в зависимости от статус кода
	switch resp.StatusCode {
	case 200:
		var successResp SuccessResponse
		if err := json.Unmarshal(body, &successResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга успешного ответа: %w", err)
		}
		fmt.Println("количество найденных ошибок: ", len(successResp.Errors))
		fmt.Println("ошибки в ответе от ллм: ", successResp.Errors)
		result.Success = &successResp

	case 422:
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга ответа об ошибке: %w", err)
		}
		result.Error = &errorResp

	default:
		return nil, fmt.Errorf("неожиданный статус код: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

	return result, nil
}

// Пример использования
func (c *Client) Analyze(doc string) (*SuccessResponse, error) {
	// Создаем запрос
	req := Request{
		HTML: doc,
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
