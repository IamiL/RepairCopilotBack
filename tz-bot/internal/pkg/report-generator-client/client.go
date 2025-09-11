package reportgeneratorclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	tzservice "repairCopilotBot/tz-bot/internal/service/tz"
	"time"
)

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Client struct {
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

func New(host string, port int) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: fmt.Sprintf("%s:%d", host, port),
	}
}

type ErrorsArray struct {
	Errors []tzservice.Error `json:"errors"`
}

func (c *Client) GenerateDocument(ctx context.Context, errors []tzservice.Error) ([]byte, error) {

	errorsArray := ErrorsArray{
		Errors: errors,
	}
	//for _, v := range errorsArray.Errors {
	//	if v.MissingInstances != nil && len(*v.MissingInstances) > 0 {
	//		if v.InvalidInstances == nil {
	//			invInsts := make([]OutInvalidError, 0)
	//			v.InvalidInstances = &invInsts
	//		}
	//		for j := range *v.MissingInstances {
	//			*v.InvalidInstances = append(*v.InvalidInstances, OutInvalidError{
	//				SuggestedFix: (*v.MissingInstances)[j].SuggestedFix,
	//				Quote:        "...",
	//			})
	//		}
	//
	//	}
	//}
	// Сериализуем ErrorsArray в JSON
	jsonData, err := json.Marshal(errorsArray)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal errors array: %w", err)
	}
	fmt.Println(string(jsonData))
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
	_, err = c.extractFilename(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return nil, fmt.Errorf("failed to extract filename: %w", err)
	}

	return body, nil
}

func (c *Client) extractFilename(contentDisposition string) (string, error) {
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
