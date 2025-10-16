package llmClient

import (
	"regexp"
	"strings"
)

// MarkdownToHTML converts markdown formatted text to HTML
func MarkdownToHTML(markdown string) string {
	html := markdown

	// Convert bold text: **text** -> <strong>text</strong>
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	html = boldRegex.ReplaceAllString(html, "<strong>$1</strong>")

	// Convert italic text: *text* -> <em>text</em>
	italicRegex := regexp.MustCompile(`\*([^*]+)\*`)
	html = italicRegex.ReplaceAllString(html, "<em>$1</em>")

	// Split by lines to process lists
	lines := strings.Split(html, "\n")
	var result []string
	inList := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if line is a list item
		if strings.HasPrefix(trimmedLine, "- ") {
			if !inList {
				result = append(result, "<ul>")
				inList = true
			}
			// Remove "- " and wrap in <li>
			listItem := strings.TrimPrefix(trimmedLine, "- ")
			result = append(result, "  <li>"+listItem+"</li>")
		} else {
			// If we were in a list, close it
			if inList {
				result = append(result, "</ul>")
				inList = false
			}

			// Add the line (add <br> if not empty and not the last line)
			if trimmedLine != "" {
				if i < len(lines)-1 && lines[i+1] != "" && !strings.HasPrefix(strings.TrimSpace(lines[i+1]), "- ") {
					result = append(result, line+"<br>")
				} else {
					result = append(result, line)
				}
			} else {
				result = append(result, "")
			}
		}
	}

	// Close list if still open
	if inList {
		result = append(result, "</ul>")
	}

	return strings.Join(result, "\n")
}