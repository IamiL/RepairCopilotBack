package word_parser2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// WordConverterClient represents a client for the word converter service
type WordConverterClient struct {
	Host   string
	Port   int
	client *http.Client
}

// ConvertResponse represents the response from the convert endpoint
type ConvertResponse struct {
	HTML string `json:"html"`
	CSS  string `json:"css"`
}

// ConvertResponseWithParagraphs represents the extended response with extracted paragraphs
type ConvertResponseWithParagraphs struct {
	HTML       string `json:"html"`
	CSS        string `json:"css"`
	Paragraphs string `json:"paragraphs"`
}

// ParagraphExtractionResult represents the result of paragraph extraction
type ParagraphExtractionResult struct {
	HTMLWithPlaceholder string
	Paragraphs          string
}

// NewWordConverterClient creates a new instance of WordConverterClient
func NewWordConverterClient(host string, port int) *WordConverterClient {
	return &WordConverterClient{
		Host: host,
		Port: port,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Convert uploads a DOCX file and returns the HTML with placeholder, CSS content, and extracted paragraphs
func (c *WordConverterClient) Convert(fileData []byte, filename string) (string, string, error) {
	// Create a buffer to store the multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create a form file field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", "", fmt.Errorf("failed to create form file: %w", err)
	}

	// Write the file data to the form
	_, err = part.Write(fileData)
	if err != nil {
		return "", "", fmt.Errorf("failed to write file data: %w", err)
	}

	// Close the writer to finalize the form data
	err = writer.Close()
	if err != nil {
		return "", "", fmt.Errorf("failed to close form writer: %w", err)
	}

	// Create the HTTP request
	url := fmt.Sprintf("http://%s:%d/convert", c.Host, c.Port)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set the content type header
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var convertResp ConvertResponse
	err = json.Unmarshal(body, &convertResp)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Replace escaped quotes with regular quotes in HTML
	cleanedHTML := strings.ReplaceAll(convertResp.HTML, `\"`, `"`)

	// Extract paragraphs from HTML
	//extraction := extractParagraphs(cleanedHTML)

	return cleanedHTML, convertResp.CSS, nil
}

const paragraphPlaceholder = "{{PARAGRAPHS_PLACEHOLDER}}"

// extractParagraphs extracts paragraphs from HTML and replaces them with a placeholder
func ExtractParagraphs(html string) ParagraphExtractionResult {
	// Regular expression to find the article content with paragraphs (with possible attributes)
	articleRegex := regexp.MustCompile(`<article[^>]*>(.*?)</article>`)
	matches := articleRegex.FindStringSubmatch(html)

	if len(matches) < 2 {
		return ParagraphExtractionResult{
			HTMLWithPlaceholder: html,
			Paragraphs:          "",
		}
	}

	paragraphs := matches[1]
	htmlWithPlaceholder := articleRegex.ReplaceAllString(html, "<article>"+paragraphPlaceholder+"</article>")

	fmt.Println("возвращаем из extractParagraphs: paragraphs = ", paragraphs)

	return ParagraphExtractionResult{
		HTMLWithPlaceholder: htmlWithPlaceholder,
		Paragraphs:          paragraphs,
	}
}

// InsertParagraphs inserts paragraphs back into HTML replacing the placeholder
func InsertParagraphs(htmlWithPlaceholder, paragraphs string) string {
	return regexp.MustCompile(regexp.QuoteMeta(paragraphPlaceholder)).ReplaceAllString(htmlWithPlaceholder, paragraphs)
}

// Health checks if the service is healthy
func (c *WordConverterClient) Health() error {
	url := fmt.Sprintf("http://%s:%d/health", c.Host, c.Port)
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service is not healthy, status: %d", resp.StatusCode)
	}

	return nil
}

// Example usage
func main() {
	// Initialize the client
	client := NewWordConverterClient("localhost", 3000)

	// Check if service is healthy
	err := client.Health()
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}
	fmt.Println("Service is healthy")

	// Example usage (commented out as we don't have an actual file)
	/*
		// Read a DOCX file
		fileData, err := os.ReadFile("example.docx")
		if err != nil {
			fmt.Printf("Failed to read file: %v\n", err)
			return
		}

		// Convert the file
		html, css, paragraphs, err := client.Convert(fileData, "example.docx")
		if err != nil {
			fmt.Printf("Conversion failed: %v\n", err)
			return
		}

		fmt.Printf("HTML length: %d\n", len(html))
		fmt.Printf("CSS length: %d\n", len(css))
		fmt.Printf("Paragraphs length: %d\n", len(paragraphs))
		fmt.Println("Conversion successful!")
	*/
}
