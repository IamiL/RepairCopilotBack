package tz_llm_client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	modelrepo "repairCopilotBot/tz-bot/internal/repository/models"
	"time"

	"repairCopilotBot/tz-bot/internal/repository"

	"github.com/google/uuid"
)

// LLMCacheRepository defines the interface for LLM cache operations
type LLMCacheRepository interface {
	// GetCachedResponse retrieves cached response by messages hash
	GetCachedResponse(ctx context.Context, messagesHash string) (*modelrepo.LLMCache, error)

	// SaveCachedResponse saves a new cached response
	SaveCachedResponse(ctx context.Context, req *modelrepo.CreateLLMCacheRequest) (*modelrepo.LLMCache, error)
}
type Config struct {
	Url   string `yaml:"url"`
	Model string `yaml:"model"`
}

type Client struct {
	url        string // URL API для отправки файла (замените на реальный URL) apiURL := "https://your-api-endpoint.com/upload"
	model      string
	repository LLMCacheRepository
}

func New(url string, model string) *Client {
	return &Client{url: url, model: model}
}

func NewWithCache(url string, model string, repo LLMCacheRepository) *Client {
	return &Client{url: url, model: model, repository: repo}
}

// Request структура для отправки запроса
type Request struct {
	Mode     string `json:"mode"`
	Model    string `json:"model"`
	Messages []struct {
		Role    *string `json:"role"`
		Content *string `json:"content"`
	} `json:"messages"`
	//Schema map[string]interface{} `json:"schema"`
	//Codes []string `json:"codes"`
}

type SuccessResponse struct {
	Result *GroupReport `json:"result"`
	Usage  *struct {
		PromptTokens     *int `json:"prompt_tokens"`
		CompletionTokens *int `json:"completion_tokens"`
		TotalTokens      *int `json:"total_tokens"`
	} `json:"usage"`
	Cost *struct {
		Currency   *string `json:"currency"`
		ModelLabel *string `json:"model_label"`
		Mode       *string `json:"mode"`
		//PricePer1M *int     `json:"price_per_1m"`
		TotalRub *float64 `json:"total_rub"`
	} `json:"cost"`
	ModelUri *string `json:"model_uri"`
	Attempts *int    `json:"attempts"`
}

type GroupReport struct {
	GroupID          *string        `json:"group_id"`
	Errors           *[]ErrorReport `json:"errors"`
	PreliminaryNotes *string        `json:"preliminary_notes"`
	OverallCritique  *string        `json:"overall_critique"`
}

type ErrorReport struct {
	ID        uuid.UUID   `json:"id"`
	Code      *string     `json:"code"`
	Instances *[]Instance `json:"instances"`
	Process   *Process    `json:"process"`
	Verdict   *string     `json:"verdict"`
}

type Process struct {
	Analysis     *string      `json:"analysis"`
	Critique     *string      `json:"critique"`
	Verification *string      `json:"verification"`
	Retrieval    *[]Retrieval `json:"retrieval"`
}

type Instance struct {
	ErrType      *string `json:"err_type"`
	Snippet      *string `json:"snippet"`
	LineStart    *int    `json:"line_start"`
	LineEnd      *int    `json:"line_end"`
	SuggestedFix *string `json:"suggested_fix"`
	Rationale    *string `json:"rationale"`
}

type Retrieval struct {
	Text      *string `json:"text"`
	LineStart *int    `json:"line_start"`
	LineEnd   *int    `json:"line_end"`
}

// calculateMessagesHash вычисляет SHA-256 хэш от массива сообщений
func calculateMessagesHash(messages []struct {
	Role    *string `json:"role"`
	Content *string `json:"content"`
}) (string, error) {
	// Сериализуем массив сообщений в JSON для получения стабильного хэша
	jsonData, err := json.Marshal(messages)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации сообщений для хэша: %w", err)
	}

	// Вычисляем SHA-256 хэш
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}

// MakeHTTPRequest выполняет реальный HTTP запрос к LLM API
func (c *Client) makeHTTPRequest(req Request) (*SuccessResponse, error) {
	// Сериализуем запрос в JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}
	fmt.Println("JSON DATA В ЗАПРОСЕ К ЛЛМ НАЧАЛО:")
	fmt.Println(string(jsonData))
	fmt.Println("--------КОНЕЦ-----------")

	fmt.Printf("Отправляем запрос к LLM API: %s\n", c.url)

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
		var errorResp interface{}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга ответа с ошибкой валидации: %w", err)
		}
		return nil, fmt.Errorf("ошибочный ответ от ллм реквестера - %#v", errorResp)

	default:
		// Другие коды ответов
		return nil, fmt.Errorf("неожиданный статус код: %d, тело ответа: %s", resp.StatusCode, string(body))
	}

}

func (c *Client) SendMessage(Messages []struct {
	Role    *string `json:"role"`
	Content *string `json:"content"`
},
	Schema map[string]interface{},
) (*SuccessResponse, error) {
	ctx := context.Background()

	// Если репозиторий настроен, проверяем кэш
	if c.repository != nil {
		// Вычисляем хэш сообщений
		messagesHash, err := calculateMessagesHash(Messages)
		if err != nil {
			// Если не удалось вычислить хэш, продолжаем без кэша
			fmt.Printf("Ошибка вычисления хэша сообщений: %v\n", err)
		} else {
			// Проверяем наличие ответа в кэше
			cachedResponse, err := c.repository.GetCachedResponse(ctx, messagesHash)
			if err == nil && cachedResponse != nil {
				// Десериализуем кэшированный ответ
				var response SuccessResponse
				if err := json.Unmarshal(cachedResponse.ResponseData, &response); err == nil {
					fmt.Printf("Найден кэшированный ответ для запроса\n")
					return &response, nil
				} else {
					fmt.Printf("Ошибка десериализации кэшированного ответа: %v\n", err)
				}
			} else if !errors.Is(err, repository.ErrLLMCacheNotFound) {
				fmt.Printf("Ошибка проверки кэша: %v\n", err)
			}
		}
	}

	// Создаем запрос
	req := Request{
		Mode:     "sync",
		Model:    c.model,
		Messages: Messages,
		//Schema:   Schema,
	}

	// Выполняем запрос
	response, err := c.makeHTTPRequest(req)
	if err != nil {

		fmt.Println(" отладка 11")
		return nil, err
	}

	fmt.Println(" отладка 12")

	// Если репозиторий настроен, сохраняем ответ в кэш
	if c.repository != nil && response != nil {
		messagesHash, hashErr := calculateMessagesHash(Messages)
		if hashErr == nil {
			responseData, marshalErr := json.Marshal(response)
			if marshalErr == nil {
				now := time.Now()
				cacheReq := &modelrepo.CreateLLMCacheRequest{
					MessagesHash: messagesHash,
					ResponseData: responseData,
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				_, cacheErr := c.repository.SaveCachedResponse(ctx, cacheReq)
				if cacheErr != nil {
					fmt.Printf("Ошибка сохранения ответа в кэш: %v\n", cacheErr)
				} else {
					fmt.Printf("Ответ успешно сохранён в кэш\n")
				}
			} else {
				fmt.Printf("Ошибка сериализации ответа для кэша: %v\n", marshalErr)
			}
		} else {
			fmt.Printf("Ошибка вычисления хэша для сохранения в кэш: %v\n", hashErr)
		}
	}

	return response, nil
}
