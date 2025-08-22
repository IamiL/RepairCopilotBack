package tzservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

type DocxGeneratorClient struct {
	httpClient *http.Client
	baseURL    string
}

type ValidationError struct {
	Detail []ValidationDetail `json:"detail"`
}

type ValidationDetail struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %+v", e.Detail)
}

type GenerateResponse struct {
	Data     []byte
	Filename string
}

func NewClient(baseURL string, timeout time.Duration) *DocxGeneratorClient {
	return &DocxGeneratorClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

func (c *DocxGeneratorClient) GenerateDocument(ctx context.Context, errorsArray ErrorsArray) (*GenerateResponse, error) {
	// Сериализуем ErrorsArray в JSON
	jsonData, err := json.Marshal(errorsArray)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal errors array: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/generate-report", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Обрабатываем ошибки
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnprocessableEntity {
			var validationErr ValidationError
			if unmarshalErr := json.Unmarshal(body, &validationErr); unmarshalErr == nil {
				return nil, validationErr
			}
		}
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Проверяем Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}

	// Извлекаем filename из Content-Disposition
	filename, err := c.extractFilename(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return nil, fmt.Errorf("failed to extract filename: %w", err)
	}

	return &GenerateResponse{
		Data:     body,
		Filename: filename,
	}, nil
}

func (c *DocxGeneratorClient) extractFilename(contentDisposition string) (string, error) {
	if contentDisposition == "" {
		return "", fmt.Errorf("content-disposition header is empty")
	}

	// Регулярное выражение для извлечения filename из Content-Disposition
	re := regexp.MustCompile(`filename="([^"]+)"`)
	matches := re.FindStringSubmatch(contentDisposition)

	if len(matches) < 2 {
		return "", fmt.Errorf("filename not found in content-disposition: %s", contentDisposition)
	}

	return matches[1], nil
}
