package tui

import (
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"

	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/provider"
)

// renderMarkdown renders source as styled terminal output using
// glamourStyle(theme), so it always matches the active theme - built-in
// or a fully custom one from Lua - rather than one of glamour's own fixed
// bundled styles. Used for both the message preview pane and compose's
// live preview, so themeing and behavior stay identical between reading
// and writing mail.
func renderMarkdown(theme config.Theme, width int, source string) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	if width < 1 {
		width = 1
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyle(theme)),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return source
	}
	out, err := r.Render(source)
	if err != nil {
		return source
	}
	return strings.TrimRight(out, "\n")
}

// renderMessageBody renders a message for display. HTML mail is converted
// to Markdown first (it's rarely hand-authored Markdown-compatible prose,
// but this reads far better than raw tags), then both paths go through
// the same theme-aware renderer as compose's preview.
func renderMessageBody(theme config.Theme, width int, msg provider.Message) string {
	if msg.BodyHTML != "" {
		if markdown, err := htmltomarkdown.ConvertString(msg.BodyHTML); err == nil {
			return renderMarkdown(theme, width, markdown)
		}
	}
	return renderMarkdown(theme, width, msg.BodyText)
}

// glamourStyle derives a glamour style config directly from theme's
// colors, so any theme - built-in or a user's custom Lua one - renders
// Markdown/HTML content consistently with the rest of the UI, rather than
// requiring its own separate glamour style to keep in sync.
func glamourStyle(theme config.Theme) ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Fg)},
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Muted), Italic: boolPtr(true)},
			Indent:         uintPtr(1),
			IndentToken:    strPtr("│ "),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Fg)},
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Fg)},
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Accent), Bold: boolPtr(true)},
		},
		Strong:        ansi.StylePrimitive{Color: strPtr(theme.Fg), Bold: boolPtr(true)},
		Emph:          ansi.StylePrimitive{Color: strPtr(theme.Fg), Italic: boolPtr(true)},
		Strikethrough: ansi.StylePrimitive{CrossedOut: boolPtr(true)},
		HorizontalRule: ansi.StylePrimitive{
			Color:  strPtr(theme.Border),
			Format: "\n────────\n",
		},
		Item:        ansi.StylePrimitive{Color: strPtr(theme.Fg)},
		Enumeration: ansi.StylePrimitive{Color: strPtr(theme.Fg)},
		Link:        ansi.StylePrimitive{Color: strPtr(theme.Accent), Underline: boolPtr(true)},
		LinkText:    ansi.StylePrimitive{Color: strPtr(theme.Accent)},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Fg), BackgroundColor: strPtr(theme.SelectBg)},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Fg), BackgroundColor: strPtr(theme.SelectBg)},
				Margin:         uintPtr(1),
			},
		},
		Table: ansi.StyleTable{
			StyleBlock:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: strPtr(theme.Fg)}},
			CenterSeparator: strPtr("┼"),
			ColumnSeparator: strPtr("│"),
			RowSeparator:    strPtr("─"),
		},
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func uintPtr(u uint) *uint    { return &u }
