package provider

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	html, err := RenderMarkdown("**hi** _there_")
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	if !strings.Contains(html, "<strong>hi</strong>") || !strings.Contains(html, "<em>there</em>") {
		t.Fatalf("html = %q, want <strong>/<em> tags", html)
	}
}
