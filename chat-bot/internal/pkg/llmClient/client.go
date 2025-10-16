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

// MessagePair represents a pair of user and bot messages in history
type MessagePair struct {
	User string `json:"user"`
	Bot  string `json:"bot"`
}

// ChatState holds the current state of the dialog (history and tree)
type ChatState struct {
	History []MessagePair          `json:"history"`
	Tree    map[string]interface{} `json:"tree"`
}

// ChatRequest is the request body for /search endpoint
type ChatRequest struct {
	UserMessage string                 `json:"user_message"`
	History     []MessagePair          `json:"history"`
	Tree        map[string]interface{} `json:"tree"`
}

// ChatResponse is the response body from /search endpoint
type ChatResponse struct {
	Response string                 `json:"response"`
	Tree     map[string]interface{} `json:"tree"`
}

// EndDialogRequest is the request body for /end_dialog endpoint
type EndDialogRequest struct {
	History []MessagePair          `json:"history"`
	Tree    map[string]interface{} `json:"tree"`
}

// EndDialogResponse is the response body from /end_dialog endpoint
type EndDialogResponse struct {
	Summary string `json:"summary"`
}

// SendMessage sends a message to the LLM service and returns the response and updated state
// For new dialogs, pass an empty ChatState with History: []MessagePair{} and Tree: make(map[string]interface{})
func (c *Client) SendMessage(state *ChatState, message string) (string, *ChatState, error) {
	if c.Mocked {
		time.Sleep(time.Second * 5)
		mockResponse := "замокано (ответ на сообщение - " + message + ")"
		newState := &ChatState{
			History: append(state.History, MessagePair{User: message, Bot: mockResponse}),
			Tree:    state.Tree,
		}
		return mockResponse, newState, nil
	}

	requestBody := ChatRequest{
		UserMessage: message,
		History:     state.History,
		Tree:        state.Tree,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	endpoint := c.Url + "/chat"
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Error: %v", endpoint, string(jsonData), err)
		return "", nil, fmt.Errorf("error sending POST request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Error reading response: %v", endpoint, string(jsonData), statusCode, err)
		return "", nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnprocessableEntity { // 422
		var validationErr ValidationError
		if err := json.Unmarshal(body, &validationErr); err != nil {
			log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s, Error parsing validation: %v", endpoint, string(jsonData), resp.StatusCode, string(body), err)
			return "", nil, fmt.Errorf("failed to parse validation error: %w", err)
		}
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s, Validation error: %v", endpoint, string(jsonData), resp.StatusCode, string(body), validationErr)
		return "", nil, validationErr
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s", endpoint, string(jsonData), resp.StatusCode, string(body))
		return "", nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var response ChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Update state with the new message pair and tree
	newState := &ChatState{
		History: append(state.History, MessagePair{User: message, Bot: response.Response}),
		Tree:    response.Tree,
	}

	return response.Response, newState, nil
}

// FinishChat ends the dialog and returns a summary
func (c *Client) FinishChat(state *ChatState) (string, error) {
	if c.Mocked {
		time.Sleep(time.Second * 5)
		return "замокано (завершение чата)", nil
	}

	requestBody := EndDialogRequest{
		History: state.History,
		Tree:    state.Tree,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	endpoint := c.Url + "/end_dialog"
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Error: %v", endpoint, string(jsonData), err)
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Error reading response: %v", endpoint, string(jsonData), statusCode, err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnprocessableEntity { // 422
		var validationErr ValidationError
		if err := json.Unmarshal(body, &validationErr); err != nil {
			log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s, Error parsing validation: %v", endpoint, string(jsonData), resp.StatusCode, string(body), err)
			return "", fmt.Errorf("failed to parse validation error: %w", err)
		}
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s, Validation error: %v", endpoint, string(jsonData), resp.StatusCode, string(body), validationErr)
		return "", validationErr
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[LLM Client Error] Endpoint: %s, Method: POST, Request body: %s, Response status: %d, Response body: %s", endpoint, string(jsonData), resp.StatusCode, string(body))
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var response EndDialogResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Summary, nil
}
