package docxToDocx2007clientclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Client — HTTP-клиент для сервиса конвертации DOCX → DOCX (2007).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option — функциональные опции для конфигурации клиента.
type Option func(*Client)

// WithHTTPClient позволяет подменить http.Client (например, задать прокси/транспорт).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithScheme меняет схему ("http" или "https"), по умолчанию — "http".
func WithScheme(scheme string) Option {
	return func(c *Client) {
		s := strings.ToLower(strings.TrimSpace(scheme))
		if s != "http" && s != "https" {
			s = "http"
		}
		// baseURL будет пересобран в New()
		c.baseURL = s
	}
}

// WithBasePath задаёт базовый путь сервиса (например, "/api").
// По умолчанию — пусто.
func WithBasePath(path string) Option {
	// временно сохраняем в baseURL хвостом; в New() пересоберём корректно
	return func(c *Client) { c.baseURL = strings.TrimRight(c.baseURL, "/") + "|" + "/" + strings.Trim(path, "/") }
}

// New создаёт клиента. Обязательно укажите host и port.
// По умолчанию используется схема http, таймаут 60s.
func New(host string, port int, opts ...Option) (*Client, error) {
	if strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("host пуст")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("port некорректен: %d", port)
	}

	c := &Client{
		// httpClient по умолчанию с разумным таймаутом
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	// Применяем опции (схема/путь могут временно сохраняться в baseURL)
	for _, opt := range opts {
		opt(c)
	}

	// Разбираем схему и basePath, если они были заданы опциями
	scheme := "http"
	basePath := ""
	if c.baseURL != "" {
		parts := strings.SplitN(c.baseURL, "|", 2)
		switch len(parts) {
		case 1:
			if parts[0] == "http" || parts[0] == "https" {
				scheme = parts[0]
			}
		case 2:
			if parts[0] == "http" || parts[0] == "https" {
				scheme = parts[0]
			}
			basePath = strings.Trim(parts[1], "/")
		}
	}

	if basePath != "" {
		c.baseURL = fmt.Sprintf("%s://%s:%d/%s", scheme, host, port, basePath)
	} else {
		c.baseURL = fmt.Sprintf("%s://%s:%d", scheme, host, port)
	}

	return c, nil
}

// Convert отправляет один DOCX-файл в сервис и возвращает сконвертированный файл.
// filename — исходное имя файла (используется в multipart форме).
// Возвращает []byte (docx или zip — в зависимости от сервиса; при одном файле это .docx) и error.
func (c *Client) Convert(ctx context.Context, file []byte, filename string) ([]byte, error) {
	if len(file) == 0 {
		return nil, fmt.Errorf("пустые данные файла")
	}
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("пустое имя файла")
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// Поле "files" — как ожидает сервер FastAPI.
	part, err := w.CreateFormFile("files", filepath.Base(filename))
	if err != nil {
		return nil, fmt.Errorf("multipart create: %w", err)
	}
	if _, err = part.Write(file); err != nil {
		return nil, fmt.Errorf("multipart write: %w", err)
	}
	if err = w.Close(); err != nil {
		return nil, fmt.Errorf("multipart close: %w", err)
	}

	url := c.baseURL + "/convert"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	// При отправке одного файла логично просить .docx
	req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.wordprocessingml.document, application/zip, */*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ целиком
	respBytes, readErr := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// Включим часть тела ошибки для диагностики
		msg := strings.TrimSpace(string(respBytes))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("convert failed: status=%d: %s", resp.StatusCode, msg)
	}
	if readErr != nil {
		return nil, fmt.Errorf("read body: %w", readErr)
	}

	return respBytes, nil
}

/*
Пример использования:

package main

import (
	"context"
	"os"
	"time"

	"github.com/your/module/path/docxclient"
)

func main() {
	cli, err := docxclient.New("localhost", 8000)
	if err != nil {
		panic(err)
	}

	src, err := os.ReadFile("input.docx")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	out, err := cli.Convert(ctx, src, "input.docx")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile("output_2007.docx", out, 0644); err != nil {
		panic(err)
	}
}
*/
