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
	Url   string `env:"URL" env-required:"true"`
	Model string `env:"MODEL" env-required:"true"`
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
	Schema json.RawMessage `json:"schema"`
	//Codes []string `json:"codes"`
}

//type SuccessResponseStep2 struct {
//	ResultStep2 *LlmReport
//}

type SuccessResponse struct {
	ResultStep2 *LlmReport      `json:"result_step_2"`
	Result      *GroupReport    `json:"result_step_1"`
	ResultRaw   json.RawMessage `json:"result"`
	Usage       *struct {
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
	Duration *int64  `json:"duration"`
}

type GroupReport struct {
	GroupID    *int           `json:"group_id"`
	GroupTitle *string        `json:"group_title"`
	Errors     *[]ErrorReport `json:"errors"`
	//PreliminaryNotes *string        `json:"preliminary_notes"`
	//OverallCritique  *string        `json:"overall_critique"`
}

type ErrorReport struct {
	ID            uuid.UUID `json:"id"`
	Code          *string   `json:"error_id"`
	Title         *string   `json:"title"`
	AnalysisSteps *[]struct {
		Goal     *string `json:"goal"`
		Observed *string `json:"observed"`
	} `json:"analysis_steps"`
	//AnalysisLines []int   `json:"analysis_lines"`
	Critique *string `json:"critique"`
	Verdict  struct {
		TextVerdict string `json:"text_verdict"`
		Status      string `json:"status"`
	} `json:"verdict"`
	Instances *[]Instance `json:"instances"`
	//Process   *Process    `json:"process"`
}

//type Process struct {
//	Analysis     *string      `json:"analysis"`
//	Critique     *string      `json:"critique"`
//	Verification *string      `json:"verification"`
//	Retrieval    *[]Retrieval `json:"retrieval"`
//}

type Instance struct {
	LlmId           *string  `json:"id"`
	Fix             *string  `json:"fix"`
	Kind            *string  `json:"kind"`
	Lines           []int    `json:"lines"`
	Risks           *string  `json:"risks"`
	Quotes          []string `json:"quotes"`
	Priority        *string  `json:"priority"`
	Sections        []string `json:"sections"`
	WhatIsIncorrect *string  `json:"what_is_incorrect"`
	//ErrType      *string `json:"err_type"`
	//Snippet         *string  `json:"snippet"`
	//LineStart       *int     `json:"line_start"`
	//LineEnd         *int     `json:"line_end"`
	//SuggestedFix    *string  `json:"suggested_fix"`
	//Rationale       *string  `json:"rationale"`
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

func (c *Client) makeHTTPRequest(req Request, stepNumber int) (*SuccessResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	// 1 попытка сразу + 3 ретрая: 5с, 30с, 60с
	backoffs := []time.Duration{0, 5 * time.Second, 30 * time.Second, time.Minute}

	var lastErr error

	for attempt := 0; attempt < len(backoffs); attempt++ {
		if attempt > 0 {
			wait := backoffs[attempt]
			fmt.Printf("Повторная попытка %d из %d через %s...\n", attempt+1, len(backoffs), wait)
			time.Sleep(wait)
		}

		fmt.Printf("Отправляем запрос к LLM API (попытка %d из %d): %s\n", attempt+1, len(backoffs), c.url)

		httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("ошибка создания HTTP запроса: %w", err)
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{
			Timeout: 30 * time.Minute,
		}

		timeStart := time.Now()
		resp, err := client.Do(httpReq)
		if err != nil {
			fmt.Println("---------ОШИБКА ПРИ ВЫПОЛНЕНИИ HTTP-ЗАПРОСА К LLM-REQUESTER. URL: ", c.url)
			fmt.Println("ошибка - ", err.Error())
			fmt.Println("---------КОНЕЦ ОШИБКИ ПРИ ВЫПОЛНЕНИИ HTTP-ЗАПРОСА К LLM-REQUESTER")
			lastErr = fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
			continue
		}

		inspectionTime := time.Since(timeStart).Milliseconds()

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("ошибка чтения тела ответа: %w", err)
			continue
		}

		switch resp.StatusCode {
		case 200:
			fmt.Println("__________________________")
			fmt.Println(string(body))
			fmt.Println("__________________________")

			var successResp SuccessResponse
			if err := json.Unmarshal(body, &successResp); err != nil {
				lastErr = fmt.Errorf("ошибка парсинга успешного ответа: %w", err)
				continue
			}

			successResp.Duration = &inspectionTime

			if stepNumber == 1 {
				if err := json.Unmarshal(successResp.ResultRaw, &successResp.Result); err != nil {
					lastErr = fmt.Errorf("llmSendMessageMakeHTTPReq ошибка парсинга resultRawJson успешного ответа step1: %w", err)
					continue
				}
			} else if stepNumber == 2 {
				fmt.Println(string(successResp.ResultRaw))
				if err := json.Unmarshal(successResp.ResultRaw, &successResp.ResultStep2); err != nil {
					lastErr = fmt.Errorf("llmSendMessageMakeHTTPReq ошибка парсинга resultRawJson успешного ответа step2: %w", err)
					continue
				}
			}

			return &successResp, nil

		case 422:
			var errorResp interface{}
			if err := json.Unmarshal(body, &errorResp); err != nil {
				lastErr = fmt.Errorf("ошибка парсинга ответа с ошибкой валидации: %w", err)
				continue
			}
			lastErr = fmt.Errorf("ошибочный ответ от ллм реквестера - %#v", errorResp)
			// по требованию — ретраем даже такие ошибки
			continue

		default:
			lastErr = fmt.Errorf("неожиданный статус код: %d, тело ответа: %s", resp.StatusCode, string(body))
			continue
		}
	}

	return nil, fmt.Errorf("не удалось получить корректный ответ от LLM API после %d попыток: %w", len(backoffs), lastErr)
}

func (c *Client) SendMessage(Messages []struct {
	Role    *string `json:"role"`
	Content *string `json:"content"`
},
	Schema json.RawMessage,
	stepNumber int,
	useCache bool,
) (*SuccessResponse, error) {
	ctx := context.Background()
	if Schema == nil {
		return nil, errors.New("schema is null")
	}

	// Если репозиторий настроен, проверяем кэш
	if c.repository != nil && useCache {
		// Вычисляем хэш сообщений
		messagesHash, err := calculateMessagesHash(Messages)
		if err != nil {
			// Если не удалось вычислить хэш, продолжаем без кэша
			fmt.Printf("Ошибка вычисления хэша сообщений: %v\n", err)
		} else {
			// Проверяем наличие ответа в кэше
			cachedResponse, err := c.repository.GetCachedResponse(ctx, messagesHash)
			if err == nil && cachedResponse != nil {
				//fmt.Println("__________________________")
				//fmt.Println(string(cachedResponse.ResponseData))
				//fmt.Println("__________________________")
				// Десериализуем кэшированный ответ
				var response SuccessResponse
				if err := json.Unmarshal(cachedResponse.ResponseData, &response); err == nil {
					fmt.Printf("Найден кэшированный ответ для запроса\n")
					if response.Duration != nil {
						time.Sleep(time.Duration(*response.Duration) * time.Millisecond)
					}
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
		Schema:   Schema,
	}

	// Выполняем запрос
	response, err := c.makeHTTPRequest(req, stepNumber)
	if err != nil {

		//fmt.Println(" отладка 11")
		return nil, err
	}

	//fmt.Println(" отладка 12")

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

type LlmReport struct {
	Sections *[]Section `json:"sections"`
	Notes    *string    `json:"notes"`
	DocTitle *string    `json:"doc_title"`
}

type Section struct {
	ExistsInDoc        *bool               `json:"exists_in_doc"`
	InitialInstanceIds *[]string           `json:"initial_instance_ids"`
	FinalInstanceIds   *[]string           `json:"final_instance_ids"`
	PartName           *string             `json:"part"`
	Instances          *[]LlmStep2Instance `json:"instances"`
}

type LlmStep2Instance struct {
	ErrorID         *string `json:"error_id"`
	LlmID           *string `json:"llm_id"`
	WhatIsIncorrect *string `json:"what_is_incorrect"`
	Fix             *string `json:"how_to_fix"`
	Risks           *string `json:"risks"`
	Priority        *string `json:"priority"`
}

//type LlmReport struct {
//	Sections *[]Section `json:"sections"`
//	Notes    *string    `json:"notes"`
//}
//
//type Step2Instance struct {
//	WhatSsIncorrect *string `json:"what_is_incorrect"`
//	Fix             *string `json:"how_to_fix"`
//	ErrorID         *string `json:"error_id"`
//}
//
//type Section struct {
//	ExistsInDoc        bool        `json:"exists_in_doc"`
//	InitialInstanceIds []string    `json:"initial_instance_ids"`
//	FinalInstanceIds   []string    `json:"final_instance_ids"`
//	PartName           string      `json:"part"`
//	Instances          *[]Instance `json:"instances"`
//}
