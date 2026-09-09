package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/jamesjohnsdev/sift/internal/config"
)

type styles struct {
	pane         lipgloss.Style
	focusedPane  lipgloss.Style
	title        lipgloss.Style
	item         lipgloss.Style
	selectedItem lipgloss.Style
	muted        lipgloss.Style
	statusBar    lipgloss.Style
}

func newStyles(t config.Theme) styles {
	fg := lipgloss.Color(t.Fg)
	border := lipgloss.Color(t.Border)
	accent := lipgloss.Color(t.Accent)

	pane := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)

	return styles{
		pane:        pane,
		focusedPane: pane.BorderForeground(accent),
		title: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),
		item: lipgloss.NewStyle().
			Foreground(fg),
		selectedItem: lipgloss.NewStyle().
			Foreground(fg).
			Background(lipgloss.Color(t.SelectBg)).
			Bold(true),
		muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Muted)),
		statusBar: lipgloss.NewStyle().
			Foreground(fg).
			Background(lipgloss.Color(t.StatusBg)).
			Padding(0, 1),
	}
}
