package promt_builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Url1 string `yaml:"url1"`
	Url2 string `yaml:"url2"`
}

type Client struct {
	url1 string // URL API для отправки файла (замените на реальный URL) apiURL := "https://your-api-endpoint.com/upload"
	url2 string
}

func New(config Config) *Client {
	return &Client{url1: config.Url1,
		url2: config.Url2,
	}
}

// Request структура для отправки запроса
type Step1Request struct {
	Markdown string `json:"markdown"`
	GgID     int    `json:"ggid"`
	//Codes []string `json:"codes"`
}

type Step1SuccessResponse struct {
	GgID   *int            `json:"ggid"`
	Items  *[]Promt        `json:"items"`
	Schema json.RawMessage `json:"schema"`
	Groups []struct {
		Errors []struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Detector    string `json:"detector"`
		} `json:"errors"`
	} `json:"groups"`
}

type Promt struct {
	GroupId          *int       `json:"group_id"`
	GroupName        *string    `json:"groupName"`
	GroupDescription *string    `json:"groupDescription"`
	ErrorCodeIds     *[]int     `json:"errorCodeIds"`
	Messages         *[]Message `json:"messages"`
}

type Message struct {
	Role    *string `json:"role"`
	Content *string `json:"content"`
}

type ErrorDescription struct {
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Detector string `json:"detector"`
}

type ErrorResponse struct {
	Detail *[]struct {
		Loc  *[]interface{} `json:"loc"`
		Msg  *string        `json:"msg"`
		Type *string        `json:"type"`
	} `json:"detail"`
}

// MakeHTTPRequest выполняет реальный HTTP запрос к LLM API
func (c *Client) makeStep1HTTPRequest(req Step1Request) (*Step1SuccessResponse, error) {
	// Устанавливаем модель из конфигурации клиента

	// Сериализуем запрос в JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	//fmt.Printf("Отправляем запрос к promt-builder: %s\n", c.url1)

	//debugWriteFile("req.json", string(jsonData))

	// Создаем HTTP запрос
	httpReq, err := http.NewRequest("POST", c.url1, bytes.NewBuffer(jsonData))
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

	//debugWriteFile("resp.json", string(body))

	// Обрабатываем ответ в зависимости от статус кода
	switch resp.StatusCode {
	case 200:
		// Успешный ответ - парсим как ReportFile
		var successResp Step1SuccessResponse
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

func (c *Client) GeneratePromts(doc string, ggID int) (
	*[]Promt,
	json.RawMessage,
	map[string]ErrorDescription,
	error,
) {
	// Создаем запрос
	req := Step1Request{
		Markdown: doc,
		GgID:     ggID,
	}

	resp, err := c.makeStep1HTTPRequest(req)
	if err != nil {
		return nil, nil, nil, err
	}

	errorsMap := make(map[string]ErrorDescription)

	for i := range resp.Groups {
		for j := range resp.Groups[i].Errors {
			errorsMap[resp.Groups[i].Errors[j].Code] = ErrorDescription{
				Name:     resp.Groups[i].Errors[j].Name,
				Desc:     resp.Groups[i].Errors[j].Description,
				Detector: resp.Groups[i].Errors[j].Detector,
			}
		}
	}
	// Выполняем запрос
	return resp.Items, resp.Schema, errorsMap, nil

}

type Step2Request struct {
	Markdown     string `json:"markdown"`
	Step1Results string `json:"step1_results"`

	//Codes []string `json:"codes"`
}

func (c *Client) GenerateStep2Promts(step1Results string, markdownDocument string) (
	*[]Message,
	json.RawMessage,
	error,
) {
	// Создаем запрос
	req := Step2Request{
		Markdown:     markdownDocument,
		Step1Results: step1Results,
	}

	resp, err := c.makeStep2HTTPRequest(req)
	if err != nil {
		return nil, nil, err
	}

	//errorsMap := make(map[string]ErrorDescription)

	//for i := range resp.Groups {
	//	for j := range resp.Groups[i].Errors {
	//		errorsMap[resp.Groups[i].Errors[j].Code] = ErrorDescription{
	//			Name:     resp.Groups[i].Errors[j].Name,
	//			Desc:     resp.Groups[i].Errors[j].Description,
	//			Detector: resp.Groups[i].Errors[j].Detector,
	//		}
	//	}
	//}
	// Выполняем запрос
	return resp.Prompt.Messages, nil, nil

}

type Step2SuccessResponse struct {
	Prompt struct {
		Messages *[]Message `json:"messages"`
	} `json:"prompt"`
	Schema json.RawMessage `json:"schema"`
}

// MakeHTTPRequest выполняет реальный HTTP запрос к LLM API
func (c *Client) makeStep2HTTPRequest(req Step2Request) (*Step2SuccessResponse, error) {
	// Устанавливаем модель из конфигурации клиента

	// Сериализуем запрос в JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	//fmt.Printf("Отправляем запрос к promt-builder: %s\n", c.url2)

	//debugWriteFile("req.json", string(jsonData))

	// Создаем HTTP запрос
	httpReq, err := http.NewRequest("POST", c.url2, bytes.NewBuffer(jsonData))
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

	//debugWriteFile("resp.json", string(body))

	// Обрабатываем ответ в зависимости от статус кода
	switch resp.StatusCode {
	case 200:
		// Успешный ответ - парсим как ReportFile
		var successResp Step2SuccessResponse
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

func debugWriteFile(filename, content string) {
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("Ошибка записи дебаг файла %s: %v\n", filename, err)
	} else {
		fmt.Printf("Дебаг файл записан: %s\n", filename)
	}
}
