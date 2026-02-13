package slack

import (
	"regexp"
	"strings"
)

var (
	reH1 = regexp.MustCompile(`(?i)<h1[^>]*>(.*?)</h1>`)
	reH2 = regexp.MustCompile(`(?i)<h2[^>]*>(.*?)</h2>`)
	reH3 = regexp.MustCompile(`(?i)<h3[^>]*>(.*?)</h3>`)
	reLi = regexp.MustCompile(`(?i)<li[^>]*>(.*?)</li>`)
	reBr = regexp.MustCompile(`(?i)<br\s*/?>`)

	reBlockClose = regexp.MustCompile(`(?i)</(?:p|div|ul|ol|h[1-6]|blockquote|table|tr)>`)
	reTag        = regexp.MustCompile(`<[^>]*>`)
	reMultiSpace = regexp.MustCompile(`[^\S\n]{2,}`)
	reMultiBlank = regexp.MustCompile(`\n{3,}`)
)

// stripHTML converts HTML content to plain text.
func stripHTML(html string) string {
	if html == "" {
		return ""
	}

	s := html

	// Convert headings to markdown-style prefixes
	s = reH1.ReplaceAllString(s, "\n\n# $1\n\n")
	s = reH2.ReplaceAllString(s, "\n\n## $1\n\n")
	s = reH3.ReplaceAllString(s, "\n\n### $1\n\n")

	// Convert list items to "- " prefixed lines
	s = reLi.ReplaceAllString(s, "\n- $1")

	// Convert <br> to newline
	s = reBr.ReplaceAllString(s, "\n")

	// Add double newline after block-level closing tags
	s = reBlockClose.ReplaceAllString(s, "\n\n")

	// Remove all remaining HTML tags
	s = reTag.ReplaceAllString(s, "")

	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&apos;", "'")

	// Collapse multiple spaces (but not newlines) into one
	s = reMultiSpace.ReplaceAllString(s, " ")

	// Trim each line
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	s = strings.Join(lines, "\n")

	// Collapse 3+ consecutive newlines into 2
	s = reMultiBlank.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}
