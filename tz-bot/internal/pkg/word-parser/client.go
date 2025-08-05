package word_parser_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Config struct {
	Url string `yaml:"url"`
}

type Client struct {
	url string // URL API для отправки файла (замените на реальный URL) apiURL := "https://your-api-endpoint.com/upload"
}

func New(url string) *Client {
	return &Client{url: url}
}

type Response struct {
	Filename string `json:"filename"`
	Length   int    `json:"length"`
	Success  bool   `json:"success"`
	Text     string `json:"html"`
	Css      string `json:"css"`
}

func (c *Client) Convert(fileData []byte, filename string) (*string, *string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания form file: %v", err)
	}

	// Записываем байты файла в поле
	_, err = part.Write(fileData)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка записи файла: %v", err)
	}

	// Закрываем writer
	err = writer.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка закрытия writer: %v", err)
	}

	// Создаём HTTP запрос с query параметром format=html
	req, err := http.NewRequest("POST", c.url+"/api/v1/convert?format=html", &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Устанавливаем правильные заголовки
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	// Выполняем запрос
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("сервер вернул ошибку %d: %s", resp.StatusCode, string(body))
	}

	// Возвращаем HTML напрямую (word-parser возвращает HTML, а не JSON)
	htmlContent := string(body)

	// CSS пока возвращаем пустой, так как word-parser возвращает только HTML
	emptyCss := ""

	return &htmlContent, &emptyCss, nil
}

// CreateDocumentRequest представляет структуру запроса для создания документа
type CreateDocumentRequest struct {
	HTMLContent string            `json:"html_content"`
	Comments    map[string]string `json:"comments,omitempty"`
}

// CreateDocumentFromHTML отправляет HTTP POST запрос для создания Word документа из HTML
//
// Параметры:
//   - serverURL: URL сервера (например, "http://localhost:8000")
//   - htmlContent: HTML контент для конвертации
//   - comments: словарь примечаний (может быть nil)
//
// Возвращает:
//   - []byte: массив байтов Word документа
//   - error: ошибка, если что-то пошло не так
func (c *Client) CreateDocumentFromHTML(htmlContent string, comments map[string]string) ([]byte, error) {
	// Подготавливаем данные запроса
	request := CreateDocumentRequest{
		HTMLContent: htmlContent,
		Comments:    comments,
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 600 * time.Second,
	}

	// Формируем URL для запроса
	url := c.url + "/create"

	// Создаем POST запрос
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус код ответа
	if resp.StatusCode != http.StatusOK {
		// Читаем тело ответа для получения сообщения об ошибке
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("сервер вернул ошибку %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Читаем тело ответа (Word документ)
	documentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа сервера: %w", err)
	}

	return documentBytes, nil
}

// CreateDocumentFromHTMLWithOptions расширенная версия функции с дополнительными опциями
//
// Параметры:
//   - serverURL: URL сервера
//   - htmlContent: HTML контент
//   - comments: словарь примечаний
//   - timeout: таймаут запроса
//
// Возвращает:
//   - []byte: массив байтов Word документа
//   - error: ошибка
//func CreateDocumentFromHTMLWithOptions(serverURL, htmlContent string, comments map[string]string, timeout time.Duration) ([]byte, error) {
//	request := CreateDocumentRequest{
//		HTMLContent: htmlContent,
//		Comments:    comments,
//	}
//
//	jsonData, err := json.Marshal(request)
//	if err != nil {
//		return nil, fmt.Errorf("ошибка сериализации JSON: %w", err)
//	}
//
//	client := &http.Client{
//		Timeout: timeout,
//	}
//
//	url := serverURL + "/create"
//
//	// Создаем HTTP запрос с дополнительными заголовками
//	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
//	if err != nil {
//		return nil, fmt.Errorf("ошибка создания HTTP запроса: %w", err)
//	}
//
//	// Устанавливаем заголовки
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
//	req.Header.Set("User-Agent", "Go-DOCX-Client/1.0")
//
//	// Выполняем запрос
//	resp, err := client.Do(req)
//	if err != nil {
//		return nil, fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		bodyBytes, _ := io.ReadAll(resp.Body)
//		return nil, fmt.Errorf("сервер вернул ошибку %d: %s", resp.StatusCode, string(bodyBytes))
//	}
//
//	documentBytes, err := io.ReadAll(resp.Body)
//	if err != nil {
//		return nil, fmt.Errorf("ошибка чтения ответа сервера: %w", err)
//	}
//
//	return documentBytes, nil
//}
