package doctodocxconverterclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Client представляет клиент для сервиса конвертации DOC в DOCX
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// HealthResponse структура ответа health check
type HealthResponse struct {
	Status      string `json:"status"`
	LibreOffice string `json:"libreoffice"`
	Version     string `json:"version"`
}

// ConvertOptions опции для конвертации
type ConvertOptions struct {
	OutputFilename string        // Опциональное имя выходного файла
	Timeout        time.Duration // Таймаут для конвертации
}

// NewClient создает новый клиент для сервиса конвертации
func NewClient(host string, port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithOptions создает клиент с дополнительными настройками
func NewClientWithOptions(host string, port int, timeout time.Duration) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Convert конвертирует DOC файл в DOCX
func (c *Client) Convert(file []byte, filename string) ([]byte, error) {
	return c.ConvertWithContext(context.Background(), file, filename, nil)
}

// ConvertWithOptions конвертирует DOC файл с дополнительными опциями
func (c *Client) ConvertWithOptions(file []byte, filename string, opts *ConvertOptions) ([]byte, error) {
	return c.ConvertWithContext(context.Background(), file, filename, opts)
}

// ConvertWithContext конвертирует DOC файл с контекстом и опциями
func (c *Client) ConvertWithContext(ctx context.Context, file []byte, filename string, opts *ConvertOptions) ([]byte, error) {
	// Проверка входных данных
	if len(file) == 0 {
		return nil, fmt.Errorf("file is empty")
	}
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}

	// Создание multipart формы
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Добавление файла в форму
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(file)); err != nil {
		return nil, fmt.Errorf("failed to write file to form: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Построение URL
	url := c.baseURL + "/convert"
	if opts != nil && opts.OutputFilename != "" {
		url = fmt.Sprintf("%s?output_filename=%s", url, opts.OutputFilename)
	}

	// Создание запроса
	req, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Использование кастомного таймаута если указан
	client := c.httpClient
	if opts != nil && opts.Timeout > 0 {
		client = &http.Client{Timeout: opts.Timeout}
	}

	// Выполнение запроса
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Чтение ответа
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Проверка статуса ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("conversion failed with status %d: %s", resp.StatusCode, string(result))
	}

	return result, nil
}

// ConvertBatch конвертирует несколько файлов одновременно
func (c *Client) ConvertBatch(files map[string][]byte) (map[string]ConvertResult, error) {
	return c.ConvertBatchWithContext(context.Background(), files)
}

// ConvertResult результат конвертации одного файла в пакете
type ConvertResult struct {
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
	Message string `json:"message,omitempty"`
}

// BatchResponse ответ пакетной конвертации
type BatchResponse struct {
	Results []struct {
		Filename string `json:"filename"`
		Status   string `json:"status"`
		Output   string `json:"output,omitempty"`
		Message  string `json:"message,omitempty"`
	} `json:"results"`
}

// ConvertBatchWithContext конвертирует несколько файлов с контекстом
func (c *Client) ConvertBatchWithContext(ctx context.Context, files map[string][]byte) (map[string]ConvertResult, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	// Создание multipart формы
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Добавление всех файлов
	for filename, content := range files {
		part, err := writer.CreateFormFile("files", filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file for %s: %w", filename, err)
		}

		if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
			return nil, fmt.Errorf("failed to write file %s to form: %w", filename, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Создание запроса
	url := c.baseURL + "/convert-batch"
	req, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Выполнение запроса
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Чтение ответа
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Проверка статуса
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("batch conversion failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Парсинг JSON ответа
	var batchResp BatchResponse
	if err := json.Unmarshal(respBody, &batchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Преобразование в map для удобства
	results := make(map[string]ConvertResult)
	for _, r := range batchResp.Results {
		results[r.Filename] = ConvertResult{
			Status:  r.Status,
			Output:  r.Output,
			Message: r.Message,
		}
	}

	return results, nil
}

// HealthCheck проверяет состояние сервиса
func (c *Client) HealthCheck() (*HealthResponse, error) {
	return c.HealthCheckWithContext(context.Background())
}

// HealthCheckWithContext проверяет состояние сервиса с контекстом
func (c *Client) HealthCheckWithContext(ctx context.Context) (*HealthResponse, error) {
	url := c.baseURL + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service unhealthy, status %d: %s", resp.StatusCode, string(body))
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		return nil, fmt.Errorf("failed to parse health response: %w", err)
	}

	return &health, nil
}

// SetTimeout устанавливает таймаут для HTTP клиента
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// GetBaseURL возвращает базовый URL сервиса
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
