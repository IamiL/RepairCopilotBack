package mailerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	mailerURL   = "http://mailer-service:8080/send"
	defaultSubj = "Notification"
)

type sendReq struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

type sendResp struct {
	OK bool `json:"ok"`
}

// SendMailViaMailer отправляет текстовое письмо через локальный mailer-сервис.
func SendMailViaMailer(ctx context.Context, to, text string) error {
	reqBody := sendReq{
		To:      to,
		Subject: defaultSubj,
		Text:    text,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Клиент с таймаутом и уважением к ctx
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mailerURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post to mailer: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mailer status %d: %s", resp.StatusCode, string(body))
	}

	var r sendResp
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("decode mailer response: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("mailer responded with ok=false")
	}
	return nil
}
