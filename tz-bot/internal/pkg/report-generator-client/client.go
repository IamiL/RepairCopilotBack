package reportgeneratorclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Ошибка API с кодом и телом ответа (часто сервер присылает JSON с описанием ошибки).
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("api error: status=%d", e.StatusCode)
	}
	return fmt.Sprintf("api error: status=%d body=%s", e.StatusCode, e.Body)
}

type Client struct {
	host       string
	port       int
	httpClient *http.Client
}

// New создаёт клиент. Если httpClient == nil, будет использован клиент с таймаутом 60s.
func New(host string, port int) *Client {
	httpClient := &http.Client{Timeout: 60 * time.Second}

	return &Client{
		host:       host,
		port:       port,
		httpClient: httpClient,
	}
}

// GenerateReport отправляет payload на POST /report и возвращает бинарный .docx.
func (c *Client) GenerateReport(ctx context.Context, payload any) ([]byte, error) {
	if c == nil {
		return nil, errors.New("nil client")
	}
	// Собираем URL. Сервис из примера слушает по HTTP.
	url := fmt.Sprintf("http://%s:%d/report", c.host, c.port)

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// На всякий случай зададим Accept на docx; сервер всё равно отдаёт attachment.
	req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/octet-stream,*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Ограничим чтение тела ошибки, чтобы не тащить мегабайты в лог/ошибку.
		const limit = 8192
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, limit))
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(slurp)}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// По желанию: можно проверить Content-Type,
	// но это не обязательно — документ всё равно в байтах.
	return data, nil
}
