package telegramclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Config содержит настройки для Telegram клиента
type Config struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

// Client представляет клиент для работы с Telegram Bot API
type Client struct {
	botToken string
	chatID   string
	apiURL   string
	client   *http.Client
}

// New создает новый экземпляр Telegram клиента
func New(config Config) (*Client, error) {
	if config.BotToken == "" {
		return nil, fmt.Errorf("bot token is required")
	}
	if config.ChatID == "" {
		return nil, fmt.Errorf("chat ID is required")
	}

	return &Client{
		botToken: config.BotToken,
		chatID:   config.ChatID,
		apiURL:   fmt.Sprintf("https://api.telegram.org/bot%s", config.BotToken),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendMessage отправляет текстовое сообщение в Telegram
func (c *Client) SendMessage(text string) error {
	url := fmt.Sprintf("%s/sendMessage", c.apiURL)

	payload := map[string]interface{}{
		"chat_id":    c.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendDocument отправляет файл в Telegram
func (c *Client) SendDocument(filePath string, fileData []byte, caption string) error {
	url := fmt.Sprintf("%s/sendDocument", c.apiURL)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Добавляем chat_id
	if err := writer.WriteField("chat_id", c.chatID); err != nil {
		return fmt.Errorf("failed to write chat_id field: %w", err)
	}

	// Добавляем caption, если есть
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return fmt.Errorf("failed to write caption field: %w", err)
		}
	}

	// Добавляем файл
	part, err := writer.CreateFormFile("document", filePath)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileData); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}