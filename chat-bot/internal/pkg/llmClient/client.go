package llmClient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
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
		Mocked: true,
	}
	return client, nil
}

type ValidationError struct {
	Detail []struct {
		Loc  []string `json:"loc"`
		Msg  string   `json:"msg"`
		Type string   `json:"type"`
	} `json:"detail"`
}

func (e ValidationError) Error() string {
	var msgs []string
	for _, detail := range e.Detail {
		msgs = append(msgs, fmt.Sprintf("%s: %s (location: %v)", detail.Type, detail.Msg, detail.Loc))
	}
	return fmt.Sprintf("Validation error: %v", msgs)
}

func (c *Client) StartDialog(chatID uuid.UUID) error {
	if c.Mocked {
		time.Sleep(time.Second * 5)
		return nil
	}
	baseURL := c.Url + "/start_dialog"
	params := url.Values{}
	params.Add("user_id", chatID.String())

	resp, err := http.Post(fmt.Sprintf("%s?%s", baseURL, params.Encode()), "application/json", nil)
	if err != nil {
		fmt.Println(fmt.Errorf("request failed: %v", err))

		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to read response: %v", err))
	}
	if resp.StatusCode == http.StatusUnprocessableEntity { // 422
		var validationErr ValidationError
		if err := json.Unmarshal(body, &validationErr); err != nil {
			fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
		}
		fmt.Println(validationErr)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("Unexpected status code: " + resp.Status)
	}

	return nil
}

type ClientRequestBody struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type ClientResponseBody struct {
	Message string `json:"response"`
}

func (c *Client) SendMessage(chatId uuid.UUID, message string) (string, error) {
	if c.Mocked {
		time.Sleep(time.Second * 5)
		return "замокано (ответ на сообщение - " + message + ")", nil
	}
	requestBody := ClientRequestBody{
		UserID:  chatId.String(),
		Message: message,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return "", err
	}

	resp, err := http.Post(c.Url+"/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending POST request:", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to read response: %v", err))
	}

	var response ClientResponseBody
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
	}
	fmt.Println(response)
	fmt.Printf("Тело ответа:\n%s\n", response.Message)

	return response.Message, nil
}

type ClientResponseEndChatBody struct {
	Message string `json:"summary"`
}

func (c *Client) FinishChat(chatID uuid.UUID) (string, error) {
	if c.Mocked {
		time.Sleep(time.Second * 5)
		return "замокано (завершение чата)", nil
	}
	baseURL := c.Url + "/end_dialog"
	params := url.Values{}
	params.Add("user_id", chatID.String())

	resp, err := http.Post(fmt.Sprintf("%s?%s", baseURL, params.Encode()), "application/json", nil)
	if err != nil {
		fmt.Println(fmt.Errorf("request failed: %v", err))

		return "", err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to read response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {

		var validationErr ValidationError
		if err := json.Unmarshal(body, &validationErr); err != nil {
			fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
		}
		fmt.Println(validationErr)
		fmt.Printf("Код ответа: %d\n", resp.StatusCode)
	}

	fmt.Printf("Тело ответа:\n%s\n", string(body))

	var response ClientResponseEndChatBody
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println(fmt.Errorf("failed to parse validation error: %v", err))
	}
	fmt.Println(response)
	fmt.Printf("Тело ответа:\n%s\n", response.Message)

	return response.Message, nil
}
