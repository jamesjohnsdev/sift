// Package tui implements the Bubble Tea shell for sift: a three-pane
// tags/message-list/preview layout with vim-style navigation. Renders
// placeholder data only — no provider, storage, or config wiring yet.
package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

func Run() error {
	_, err := tea.NewProgram(newModel()).Run()
	return err
}

type focus int

const (
	focusTags focus = iota
	focusList
	focusPreview
)

type model struct {
	keys KeyMap

	tags     []tag
	messages map[string][]message

	focus      focus
	tagCursor  int
	listCursor int
	pendingG   bool

	preview viewport.Model

	width  int
	height int
}

func newModel() model {
	tags := placeholderTags()
	msgs := placeholderMessages()

	m := model{
		keys:     DefaultKeyMap(),
		tags:     tags,
		messages: msgs,
		focus:    focusTags,
		preview:  viewport.New(),
	}
	m.syncPreview()
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.preview.SetWidth(m.previewWidth())
		m.preview.SetHeight(m.paneHeight())
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// gg is the only two-key sequence; everything else clears it.
	if m.pendingG {
		m.pendingG = false
		if key.Matches(msg, m.keys.Top) {
			m.moveTop()
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Top):
		m.pendingG = true
		return m, nil

	case key.Matches(msg, m.keys.Bottom):
		m.moveBottom()
		return m, nil

	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
		return m, nil

	case key.Matches(msg, m.keys.FocusLeft):
		m.cycleFocus(-1)
		return m, nil

	case key.Matches(msg, m.keys.FocusRight):
		m.cycleFocus(1)
		return m, nil
	}
	return m, nil
}

func (m *model) cycleFocus(dir int) {
	next := int(m.focus) + dir
	if next < 0 {
		next = int(focusPreview)
	}
	if next > int(focusPreview) {
		next = int(focusTags)
	}
	m.focus = focus(next)
}

func (m *model) moveCursor(delta int) {
	switch m.focus {
	case focusTags:
		m.tagCursor = clamp(m.tagCursor+delta, 0, len(m.tags)-1)
		m.tagCursor = clampMin0(m.tagCursor)
		m.listCursor = 0
		m.syncPreview()
	case focusList:
		n := len(m.currentMessages())
		m.listCursor = clamp(m.listCursor+delta, 0, n-1)
		m.listCursor = clampMin0(m.listCursor)
		m.syncPreview()
	case focusPreview:
		if delta < 0 {
			m.preview.ScrollUp(1)
		} else {
			m.preview.ScrollDown(1)
		}
	}
}

func (m *model) moveTop() {
	switch m.focus {
	case focusTags:
		m.tagCursor = 0
		m.listCursor = 0
		m.syncPreview()
	case focusList:
		m.listCursor = 0
		m.syncPreview()
	case focusPreview:
		m.preview.GotoTop()
	}
}

func (m *model) moveBottom() {
	switch m.focus {
	case focusTags:
		m.tagCursor = clampMin0(len(m.tags) - 1)
		m.listCursor = 0
		m.syncPreview()
	case focusList:
		m.listCursor = clampMin0(len(m.currentMessages()) - 1)
		m.syncPreview()
	case focusPreview:
		m.preview.GotoBottom()
	}
}

func (m *model) currentTag() string {
	if len(m.tags) == 0 {
		return ""
	}
	return m.tags[m.tagCursor].name
}

func (m *model) currentMessages() []message {
	return m.messages[m.currentTag()]
}

func (m *model) syncPreview() {
	msgs := m.currentMessages()
	m.preview.SetContent("")
	if m.listCursor < len(msgs) {
		msg := msgs[m.listCursor]
		body := fmt.Sprintf("From: %s\nSubject: %s\nDate: %s\n\n%s",
			msg.from, msg.subject, msg.date, msg.body)
		m.preview.SetContent(body)
	}
	m.preview.GotoTop()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampMin0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
