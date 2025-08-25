package markdown_service_client

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
	url string
}

func New(url string) *Client {
	return &Client{url: url}
}

type ConvertRequest struct {
	HtmlText string `json:"html_text"`
}

type Mapping struct {
	ElementID       *string `json:"html_element_id"`
	HtmlTag         string  `json:"html_tag"`
	HtmlContent     string  `json:"html_content"`
	MarkdownStart   int     `json:"markdown_line_start"`
	MarkdownEnd     int     `json:"markdown_line_end"`
	MarkdownContent string  `json:"markdown_content"`
}

type ConvertResponse struct {
	Markdown    string    `json:"markdown"`
	Message     string    `json:"message"`
	HtmlWithIds string    `json:"html_with_ids"`
	Mappings    []Mapping `json:"mappings"`
}

// Convert отправляет HTML текст в markdown-service и получает обратно полный ответ
func (c *Client) Convert(htmlText string) (*ConvertResponse, error) {
	// Подготавливаем запрос
	request := ConvertRequest{
		HtmlText: htmlText,
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Формируем URL для запроса
	url := c.url + "/api/v1/convert"

	// Создаем POST запрос
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус код ответа
	if resp.StatusCode != http.StatusOK {
		// Читаем тело ответа для получения сообщения об ошибке
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("сервер вернул ошибку %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Читаем тело ответа
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа сервера: %w", err)
	}

	// Парсим JSON ответ
	var convertResp ConvertResponse
	err = json.Unmarshal(bodyBytes, &convertResp)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON ответа: %w", err)
	}

	return &convertResp, nil
}
