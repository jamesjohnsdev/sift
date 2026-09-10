package tui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/provider"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes escape codes so content assertions aren't tripped up by
// glamour styling multi-word text as several adjacent spans - the escape
// codes between them are invisible once rendered in a real terminal, but
// they do break a naive continuous-substring check on the raw string.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func TestRenderMarkdownEmpty(t *testing.T) {
	if got := renderMarkdown(config.TokyoNight(), 80, "   \n  "); got != "" {
		t.Fatalf("renderMarkdown(blank) = %q, want empty", got)
	}
}

func TestRenderMarkdownRendersContent(t *testing.T) {
	got := stripANSI(renderMarkdown(config.TokyoNight(), 80, "**bold** and _italic_"))
	if !strings.Contains(got, "bold") || !strings.Contains(got, "italic") {
		t.Fatalf("renderMarkdown output = %q, want the words present", got)
	}
	// Markdown syntax markers themselves shouldn't survive rendering.
	if strings.Contains(got, "**") {
		t.Fatalf("renderMarkdown output = %q, want ** markers stripped", got)
	}
}

var truecolorSeq = regexp.MustCompile(`38;2;(\d+);(\d+);(\d+)`)

// firstRGB returns the first truecolor foreground triplet found in s.
func firstRGB(t *testing.T, s string) (r, g, b int) {
	t.Helper()
	m := truecolorSeq.FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("no truecolor escape sequence found in %q", s)
	}
	r, _ = strconv.Atoi(m[1])
	g, _ = strconv.Atoi(m[2])
	b, _ = strconv.Atoi(m[3])
	return r, g, b
}

func hexRGB(t *testing.T, hex string) (r, g, b int) {
	t.Helper()
	hex = strings.TrimPrefix(hex, "#")
	rr, err := strconv.ParseInt(hex[0:2], 16, 0)
	if err != nil {
		t.Fatalf("parse hex %q: %v", hex, err)
	}
	gg, _ := strconv.ParseInt(hex[2:4], 16, 0)
	bb, _ := strconv.ParseInt(hex[4:6], 16, 0)
	return int(rr), int(gg), int(bb)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestRenderMarkdownReflectsTheme(t *testing.T) {
	source := "# Heading"
	tokyoNight := renderMarkdown(config.TokyoNight(), 80, source)
	dracula := renderMarkdown(config.Dracula(), 80, source)

	if tokyoNight == dracula {
		t.Fatal("rendering the same content under two different themes produced identical output, want theme-specific colors")
	}

	// Tokyo Night's heading should render close to its configured accent
	// color. A small tolerance absorbs harmless truecolor round-trip
	// rounding (observed off-by-one on a channel), not an exact-match
	// requirement - the point is confirming the theme's color made it
	// through at all, not pinning glamour's internal color math.
	gotR, gotG, gotB := firstRGB(t, tokyoNight)
	wantR, wantG, wantB := hexRGB(t, config.TokyoNight().Accent)
	const tolerance = 3
	if abs(gotR-wantR) > tolerance || abs(gotG-wantG) > tolerance || abs(gotB-wantB) > tolerance {
		t.Fatalf("heading color = rgb(%d,%d,%d), want close to accent rgb(%d,%d,%d)", gotR, gotG, gotB, wantR, wantG, wantB)
	}
}

func TestRenderMarkdownClampsInvalidWidth(t *testing.T) {
	// Must not panic on a width that hasn't been set yet (e.g. before the
	// first WindowSizeMsg arrives).
	got := stripANSI(renderMarkdown(config.TokyoNight(), 0, "hello"))
	if !strings.Contains(got, "hello") {
		t.Fatalf("renderMarkdown(width=0) = %q, want it to still render", got)
	}
}

func TestRenderMessageBodyConvertsHTML(t *testing.T) {
	msg := provider.Message{BodyHTML: "<p>Hello <strong>world</strong></p>"}
	got := stripANSI(renderMessageBody(config.TokyoNight(), 80, msg))

	if !strings.Contains(got, "Hello") || !strings.Contains(got, "world") {
		t.Fatalf("renderMessageBody = %q, want the text content present", got)
	}
	if strings.Contains(got, "<p>") || strings.Contains(got, "<strong>") {
		t.Fatalf("renderMessageBody = %q, want HTML tags stripped", got)
	}
}

func TestRenderMessageBodyFallsBackToBodyText(t *testing.T) {
	msg := provider.Message{BodyText: "plain text body"}
	got := stripANSI(renderMessageBody(config.TokyoNight(), 80, msg))
	if !strings.Contains(got, "plain text body") {
		t.Fatalf("renderMessageBody = %q, want the plain text content", got)
	}
}

func TestRenderMessageBodyEmpty(t *testing.T) {
	if got := renderMessageBody(config.TokyoNight(), 80, provider.Message{}); got != "" {
		t.Fatalf("renderMessageBody(empty message) = %q, want empty", got)
	}
}
