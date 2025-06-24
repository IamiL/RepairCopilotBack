package word_parser_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

type Response struct {
	Filename string `json:"filename"`
	Length   int    `json:"length"`
	Success  bool   `json:"success"`
	Text     string `json:"text"`
}

func (c *Client) Convert(file io.Reader, filename string) (*Response, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания form file: %v", err)
	}

	// Копируем содержимое файла в поле
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("ошибка копирования файла: %v", err)
	}

	// Закрываем writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("ошибка закрытия writer: %v", err)
	}

	// Создаём HTTP запрос
	req, err := http.NewRequest("POST", c.url, &buf)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Устанавливаем правильный Content-Type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	// Парсим JSON ответ
	var apiResp Response
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	return &apiResp, nil
}
