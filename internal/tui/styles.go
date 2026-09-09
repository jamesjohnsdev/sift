package tui

import "github.com/charmbracelet/lipgloss"

// Hardcoded Tokyo Night palette, standing in for the future theme system.
const (
	colorBg       = lipgloss.Color("#1a1b26")
	colorFg       = lipgloss.Color("#c0caf5")
	colorMuted    = lipgloss.Color("#565f89")
	colorBorder   = lipgloss.Color("#414868")
	colorAccent   = lipgloss.Color("#7aa2f7")
	colorSelectBg = lipgloss.Color("#283457")
	colorStatusBg = lipgloss.Color("#24283b")
)

var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	focusedPaneStyle = paneStyle.
				BorderForeground(colorAccent)

	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	itemStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(colorSelectBg).
				Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorStatusBg).
			Padding(0, 1)
)
