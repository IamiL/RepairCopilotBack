package llmClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Config struct {
	Url string `yaml:"url"`
}

type Client struct {
	Url    string
	Mocked bool
}

func New(config Config) (*Client, error) {
	client := &Client{
		Url:    config.Url,
		Mocked: false,
	}
	return client, nil
}

type ValidationError struct {
	Detail []struct {
		Loc  []interface{} `json:"loc"`
		Msg  string        `json:"msg"`
		Type string        `json:"type"`
	} `json:"detail"`
}

func (e ValidationError) Error() string {
	var msgs []string
	for _, detail := range e.Detail {
		msgs = append(msgs, fmt.Sprintf("%s: %s (location: %v)", detail.Type, detail.Msg, detail.Loc))
	}
	return fmt.Sprintf("Validation error: %v", msgs)
}

// QueryRequest is the request body for /ask endpoint
type QueryRequest struct {
	Query string `json:"query"`
}

// QueryResponse is the response body from /ask endpoint
type QueryResponse struct {
	Answer     string `json:"answer"`
	TokenCount int    `json:"token_count"`
}

// Ask sends a query to the LLM service and returns the answer and token count
func (c *Client) Ask(query string) (string, int, error) {
	if c.Mocked {
		time.Sleep(time.Second * 5)
		mockAnswer := "замокано (ответ на вопрос - " + query + ")"
		return mockAnswer, 100, nil
	}

	requestBody := QueryRequest{
		Query: query,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", 0, fmt.Errorf("error marshaling JSON: %w", err)
	}

	endpoint := c.Url + "/ask"
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Error: %v", endpoint, string(jsonData), err)
		return "", 0, fmt.Errorf("error sending POST request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Error reading response: %v", endpoint, string(jsonData), statusCode, err)
		return "", 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnprocessableEntity { // 422
		var validationErr ValidationError
		if err := json.Unmarshal(body, &validationErr); err != nil {
			log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s, Error parsing validation: %v", endpoint, string(jsonData), resp.StatusCode, string(body), err)
			return "", 0, fmt.Errorf("failed to parse validation error: %w", err)
		}
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s, Validation error: %v", endpoint, string(jsonData), resp.StatusCode, string(body), validationErr)
		return "", 0, validationErr
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s", endpoint, string(jsonData), resp.StatusCode, string(body))
		return "", 0, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var response QueryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return MarkdownToHTML(response.Answer), response.TokenCount, nil
}
