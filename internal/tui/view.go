package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const statusBarHeight = 1

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	tags := m.renderTagsPane()
	list := m.renderListPane()
	preview := m.renderPreviewPane()

	body := lipgloss.JoinHorizontal(lipgloss.Top, tags, list, preview)
	content := lipgloss.JoinVertical(lipgloss.Left, body, m.renderStatusBar())

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// Fixed 20/30/50 column split; configurable layouts come later.
func (m model) tagsWidth() int    { return m.width * 20 / 100 }
func (m model) listWidth() int    { return m.width * 30 / 100 }
func (m model) previewWidth() int { return m.width - m.tagsWidth() - m.listWidth() }

func (m model) paneHeight() int {
	h := m.height - statusBarHeight - 2 // borders
	if h < 0 {
		return 0
	}
	return h
}

func (m model) paneStyleFor(f focus, width int) lipgloss.Style {
	style := paneStyle
	if m.focus == f {
		style = focusedPaneStyle
	}
	return style.Width(width - 2).Height(m.paneHeight())
}

func (m model) renderTagsPane() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Tags") + "\n")
	for i, t := range m.tags {
		line := t.name
		if n := len(m.messages[t.name]); n > 0 {
			line = fmt.Sprintf("%s (%d)", t.name, n)
		}
		if i == m.tagCursor {
			b.WriteString(selectedItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(itemStyle.Render(line) + "\n")
		}
	}
	return m.paneStyleFor(focusTags, m.tagsWidth()).Render(b.String())
}

func (m model) renderListPane() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.currentTag()) + "\n")
	msgs := m.currentMessages()
	if len(msgs) == 0 {
		b.WriteString(mutedStyle.Render("(empty)"))
	}
	for i, msg := range msgs {
		line := fmt.Sprintf("%-20s %s", truncate(msg.from, 20), msg.subject)
		if i == m.listCursor {
			b.WriteString(selectedItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(itemStyle.Render(line) + "\n")
		}
	}
	return m.paneStyleFor(focusList, m.listWidth()).Render(b.String())
}

func (m model) renderPreviewPane() string {
	return m.paneStyleFor(focusPreview, m.previewWidth()).Render(m.preview.View())
}

func (m model) renderStatusBar() string {
	focusName := map[focus]string{
		focusTags:    "tags",
		focusList:    "list",
		focusPreview: "preview",
	}[m.focus]

	left := fmt.Sprintf(" sift  |  focus: %s", focusName)
	right := "j/k move  gg/G top/bottom  h/l focus  q quit "

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return statusBarStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
