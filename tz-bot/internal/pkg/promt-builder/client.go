package promt_builder

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
	Markdown string `json:"markdown"`
	GgID     int    `json:"ggid"`

	//Codes []string `json:"codes"`
}

type SuccessResponse struct {
	GgID   *int                   `json:"ggid"`
	Items  *[]Promt               `json:"items"`
	Schema map[string]interface{} `json:"schema"`
}

type Promt struct {
	GroupId          *int    `json:"group_id"`
	GroupName        *string `json:"groupName"`
	GroupDescription *string `json:"groupDescription"`
	ErrorCodeIds     *[]int  `json:"errorCodeIds"`
	Messages         *[]struct {
		Role    *string `json:"role"`
		Content *string `json:"content"`
	} `json:"messages"`
}

type ErrorResponse struct {
	Detail *[]struct {
		Loc  *[]interface{} `json:"loc"`
		Msg  *string        `json:"msg"`
		Type *string        `json:"type"`
	} `json:"detail"`
}

// MakeHTTPRequest выполняет реальный HTTP запрос к LLM API
func (c *Client) makeHTTPRequest(req Request) (*SuccessResponse, error) {
	// Устанавливаем модель из конфигурации клиента

	// Сериализуем запрос в JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	fmt.Printf("Отправляем запрос к promt-builder: %s\n", c.url)

	// Создаем HTTP запрос
	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	// Устанавливаем заголовки
	httpReq.Header.Set("Content-Type", "application/json")

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 30 * time.Minute, // 30 минут таймаут для LLM запросов
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

	// Обрабатываем ответ в зависимости от статус кода
	switch resp.StatusCode {
	case 200:
		// Успешный ответ - парсим как ReportFile
		var successResp SuccessResponse
		if err := json.Unmarshal(body, &successResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга успешного ответа: %w", err)
		}
		return &successResp, nil

	case 422:
		// Ошибка валидации - парсим как ErrorResponse
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга ответа с ошибкой валидации: %w", err)
		}
		errStr := ""
		for i, errorDetail := range *errorResp.Detail {
			if i > 0 {
				errStr = errStr + "\n"
			}

			if errorDetail.Msg != nil && errorDetail.Type != nil {
				errStr += "Msg: " + *errorDetail.Msg + ", Type:" + *errorDetail.Type + ", "
			}
		}

		return nil, fmt.Errorf(errStr)

	default:
		// Другие коды ответов
		return nil, fmt.Errorf("неожиданный статус код: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

}

func (c *Client) GeneratePromts(doc string, ggID int) (*SuccessResponse, error) {
	// Создаем запрос
	req := Request{
		Markdown: doc,
		GgID:     ggID,
	}

	// Выполняем запрос
	return c.makeHTTPRequest(req)

}
