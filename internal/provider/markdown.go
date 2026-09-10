package provider

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
)

// RenderMarkdown converts compose Markdown to HTML for the send path.
// Providers send both parts: the Markdown source as the plaintext
// fallback, and this HTML as the rich part.
func RenderMarkdown(markdown string) (html string, err error) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(markdown), &buf); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return buf.String(), nil
}
