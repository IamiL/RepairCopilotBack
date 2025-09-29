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
func (c *WordConverterClient) Convert(fileData []byte, filename string) (string, error) {
	// Create a buffer to store the multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create a form file field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	// Write the file data to the form
	_, err = part.Write(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to write file data: %w", err)
	}

	// Close the writer to finalize the form data
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close form writer: %w", err)
	}

	// Create the HTTP request
	url := fmt.Sprintf("http://%s:%d/convert", c.Host, c.Port)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set the content type header
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var convertResp ConvertResponse
	err = json.Unmarshal(body, &convertResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Replace escaped quotes with regular quotes in HTML
	//cleanedHTML := strings.ReplaceAll(convertResp.HTML, `\"`, `"`)

	// Extract paragraphs from HTML
	//extraction := extractParagraphs(cleanedHTML)

	return convertResp.HTML, nil
}

const paragraphPlaceholder = "{{PARAGRAPHS_PLACEHOLDER_{{ARTICLE_INDEX}}}}"

// extractChildBlocks извлекает дочерние блоки из содержимого статьи
func extractChildBlocks(articleContent string, articleIndex int) string {
	var result strings.Builder

	// Находим все прямые дочерние блоки верхного уровня
	blockRegex := regexp.MustCompile(`(?s)<(\w+)([^>]*)>(.*?)</(\w+)>`)
	blockMatches := blockRegex.FindAllStringSubmatch(articleContent, -1)

	blockIndex := 0
	for _, blockMatch := range blockMatches {
		tagName := blockMatch[1]
		existingAttrs := blockMatch[2]
		blockContent := blockMatch[3]
		closingTag := blockMatch[4]

		// Skip if opening and closing tags don't match
		if tagName != closingTag {
			continue
		}

		// Add special attributes for tracking
		newAttrs := fmt.Sprintf(`%s data-article="%d" data-block="%d"`, existingAttrs, articleIndex, blockIndex)
		processedBlock := fmt.Sprintf(`<%s%s>%s</%s>`, tagName, newAttrs, blockContent, tagName)
		result.WriteString(processedBlock)

		blockIndex++
	}

	return result.String()
}

// ExtractParagraphs extracts paragraphs from multiple articles with numbering
func ExtractParagraphs(html string) ParagraphExtractionResult {
	// Regular expression to find all article elements with their content (multiline mode)
	articleRegex := regexp.MustCompile(`(?s)<article[^>]*>(.*?)</article>`)
	articleMatches := articleRegex.FindAllStringSubmatch(html, -1)

	if len(articleMatches) == 0 {
		return ParagraphExtractionResult{
			HTMLWithPlaceholder: html,
			Paragraphs:          "",
		}
	}

	var allParagraphs strings.Builder
	htmlWithPlaceholder := html

	// Process articles in reverse order to avoid position shifts
	for i := len(articleMatches) - 1; i >= 0; i-- {
		match := articleMatches[i]
		articleContent := match[1]

		// Extract all child blocks from the article content using a more sophisticated approach
		extractedBlocks := extractChildBlocks(articleContent, i)
		allParagraphs.WriteString(extractedBlocks)

		// Extract original article attributes
		fullArticleMatch := match[0]
		articleTagRegex := regexp.MustCompile(`<article([^>]*)>`)
		articleTagMatch := articleTagRegex.FindStringSubmatch(fullArticleMatch)

		var articleAttrs string
		if len(articleTagMatch) >= 2 {
			articleAttrs = articleTagMatch[1]
		}

		// Replace this specific article with numbered placeholder, preserving attributes
		placeholder := strings.ReplaceAll(paragraphPlaceholder, "{{ARTICLE_INDEX}}", fmt.Sprintf("%d", i))
		htmlWithPlaceholder = strings.Replace(htmlWithPlaceholder, fullArticleMatch, fmt.Sprintf("<article%s>%s</article>", articleAttrs, placeholder), 1)
	}

	//fmt.Println("возвращаем из extractParagraphs: paragraphs = ", allParagraphs.String())

	return ParagraphExtractionResult{
		HTMLWithPlaceholder: htmlWithPlaceholder,
		Paragraphs:          allParagraphs.String(),
	}
}

// InsertParagraphs inserts paragraphs back into HTML replacing numbered placeholders
func InsertParagraphs(htmlWithPlaceholder, paragraphs string) string {
	// Parse all blocks from paragraphs string to organize by article
	blockRegex := regexp.MustCompile(`<(\w+)[^>]*data-article="(\d+)"[^>]*data-block="(\d+)"[^>]*>([\s\S]*?)</(\w+)>`)
	blockMatches := blockRegex.FindAllStringSubmatch(paragraphs, -1)

	// Group blocks by article index
	articleBlocks := make(map[int][]string)
	for _, match := range blockMatches {
		articleIndex := 0
		fmt.Sscanf(match[2], "%d", &articleIndex)
		blockIndex := 0
		fmt.Sscanf(match[3], "%d", &blockIndex)

		// Check if opening and closing tags match
		tagName := match[1]
		closingTag := match[5]
		if tagName != closingTag {
			continue
		}

		// Remove the special attributes from the block
		blockContent := match[4]

		// Extract original attributes by removing data-article and data-block
		fullMatch := match[0]
		startTagRegex := regexp.MustCompile(`<` + tagName + `([^>]*)>`)
		startTagMatch := startTagRegex.FindStringSubmatch(fullMatch)

		var originalAttrs string
		if len(startTagMatch) >= 2 {
			allAttrs := startTagMatch[1]
			// Remove the special tracking attributes
			allAttrs = regexp.MustCompile(`\s+data-article="[^"]*"`).ReplaceAllString(allAttrs, "")
			allAttrs = regexp.MustCompile(`\s+data-block="[^"]*"`).ReplaceAllString(allAttrs, "")
			originalAttrs = strings.TrimSpace(allAttrs)
		}

		// Reconstruct block without special attributes
		var cleanBlock string
		if originalAttrs != "" {
			cleanBlock = fmt.Sprintf(`<%s %s>%s</%s>`, tagName, originalAttrs, blockContent, tagName)
		} else {
			cleanBlock = fmt.Sprintf(`<%s>%s</%s>`, tagName, blockContent, tagName)
		}

		if articleBlocks[articleIndex] == nil {
			articleBlocks[articleIndex] = make([]string, 0)
		}

		// Insert block at correct position
		for len(articleBlocks[articleIndex]) <= blockIndex {
			articleBlocks[articleIndex] = append(articleBlocks[articleIndex], "")
		}
		articleBlocks[articleIndex][blockIndex] = cleanBlock
	}

	result := htmlWithPlaceholder

	// Replace each numbered placeholder with corresponding article content
	for articleIndex, blocks := range articleBlocks {
		placeholder := strings.ReplaceAll(paragraphPlaceholder, "{{ARTICLE_INDEX}}", fmt.Sprintf("%d", articleIndex))
		articleContent := strings.Join(blocks, "")

		result = strings.ReplaceAll(result, placeholder, articleContent)
	}

	return result
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

// Example usage (commented out for production)
/*
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
}
*/
