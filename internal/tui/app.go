// Package tui implements the Bubble Tea shell for sift: a three-pane
// tags/message-list/preview layout with vim-style navigation over data
// loaded from the local storage.Store.
package tui

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/provider"
	"github.com/jamesjohnsdev/sift/internal/storage"
)

// Run starts the TUI. updates, if non-nil, receives a pulse whenever
// background sync writes new data to storage, so the running program
// reloads instead of only ever showing what was there at startup. send
// authenticates and dispatches a composed message through the right
// account's provider.
func Run(cfg config.Config, store *storage.Store, updates <-chan struct{}, send SendFunc) error {
	m, err := newModel(cfg, store, updates, send)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m).Run()
	return err
}

type focus int

const (
	focusTags focus = iota
	focusList
	focusPreview
)

type mode int

const (
	modeInbox mode = iota
	modeCompose
)

type model struct {
	keys   KeyMap
	styles styles
	theme  config.Theme

	store   *storage.Store
	updates <-chan struct{}
	send    SendFunc

	mode    mode
	compose composeModel

	tags     []tag
	messages map[tagKey][]provider.Message

	focus      focus
	tagCursor  int
	listCursor int
	pendingG   bool

	preview viewport.Model

	width  int
	height int
}

func newModel(cfg config.Config, store *storage.Store, updates <-chan struct{}, send SendFunc) (model, error) {
	tags, messages, err := loadData(context.Background(), store)
	if err != nil {
		return model{}, err
	}

	m := model{
		keys:     newKeyMap(cfg.Keymap),
		styles:   newStyles(cfg.Theme),
		theme:    cfg.Theme,
		store:    store,
		updates:  updates,
		send:     send,
		tags:     tags,
		messages: messages,
		focus:    focusTags,
		preview:  viewport.New(),
	}
	m.syncPreview()
	return m, nil
}

// refreshMsg means the sync engine wrote new data to storage; reload it.
type refreshMsg struct{}

// listenForUpdates blocks on m.updates and turns the next pulse into a
// refreshMsg. Bubble Tea commands fire once, so this is re-issued after
// every refreshMsg to keep listening for as long as the program runs.
func listenForUpdates(updates <-chan struct{}) tea.Cmd {
	if updates == nil {
		return nil
	}
	return func() tea.Msg {
		if _, ok := <-updates; !ok {
			return nil
		}
		return refreshMsg{}
	}
}

func (m model) Init() tea.Cmd {
	return listenForUpdates(m.updates)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// The viewport has no border of its own; the outer pane style adds
		// one, so give the viewport only the content area within it.
		m.preview.SetWidth(m.previewWidth() - m.styles.pane.GetHorizontalFrameSize())
		m.preview.SetHeight(m.paneHeight() - m.styles.pane.GetVerticalFrameSize())
		// Content is rendered (word-wrapped) at SetContent time, not live by
		// the viewport, so it must be redone now that the real width is known.
		m.syncPreview()
		// m.compose is only ever constructed (via newComposeModel/newReplyModel)
		// once compose is actually opened; resizing it before that would hit an
		// uninitialized textarea.Model.
		if m.mode == modeCompose {
			m.compose.applySize(m.width, m.height)
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.mode == modeCompose {
			return m.updateCompose(msg)
		}
		return m.handleKey(msg)

	case refreshMsg:
		m.reload()
		return m, listenForUpdates(m.updates)

	case composeSentMsg:
		m.compose.sending = false
		if msg.err != nil {
			m.compose.status = "send failed: " + msg.err.Error()
			return m, nil
		}
		m.mode = modeInbox
		return m, nil
	}
	return m, nil
}

// updateCompose handles keys while the compose screen is open: a handful
// of fixed keys (not user-remappable - these are modal to the sub-screen,
// unlike the global nav bindings) plus everything else routed to whichever
// field currently has focus.
func (m model) updateCompose(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeInbox
		return m, nil

	case "ctrl+s":
		if m.compose.sending {
			return m, nil
		}
		m.compose.sending = true
		m.compose.status = "sending..."
		return m, m.compose.submit()

	case "tab":
		m.compose.field = (m.compose.field + 1) % numComposeFields
		m.compose.focusCurrent()
		return m, nil

	case "shift+tab":
		m.compose.field = (m.compose.field - 1 + numComposeFields) % numComposeFields
		m.compose.focusCurrent()
		return m, nil
	}

	var cmd tea.Cmd
	m.compose, cmd = m.compose.updateField(msg)
	return m, cmd
}

// openCompose starts a blank message addressed from the currently
// selected tag's account.
func (m *model) openCompose() {
	m.compose = newComposeModel(m.send, m.currentTagKey().account)
	m.compose.applySize(m.width, m.height)
	m.mode = modeCompose
}

// openReply starts a reply to the currently selected message, if any.
func (m *model) openReply() {
	msgs := m.currentMessages()
	if m.listCursor >= len(msgs) {
		return
	}
	m.compose = newReplyModel(m.send, m.currentTagKey().account, msgs[m.listCursor])
	m.compose.applySize(m.width, m.height)
	m.mode = modeCompose
}

// reload re-reads tags and messages from storage, trying to keep the
// current tag selected (by account+tag id, since the tag list is rebuilt
// fresh and may reorder or grow) rather than jumping back to the top.
func (m *model) reload() {
	prev := m.currentTagKey()

	tags, messages, err := loadData(context.Background(), m.store)
	if err != nil {
		return
	}
	m.tags = tags
	m.messages = messages

	m.tagCursor = 0
	for i, t := range tags {
		if t.account == prev.account && t.id == prev.id {
			m.tagCursor = i
			break
		}
	}
	m.listCursor = clampMin0(clamp(m.listCursor, 0, len(m.currentMessages())-1))
	m.syncPreview()
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

	case key.Matches(msg, m.keys.Compose):
		m.openCompose()
		return m, nil

	case key.Matches(msg, m.keys.Reply):
		m.openReply()
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

func (m *model) currentTagName() string {
	if len(m.tags) == 0 {
		return ""
	}
	return m.tags[m.tagCursor].name
}

func (m *model) currentTagKey() tagKey {
	if len(m.tags) == 0 {
		return tagKey{}
	}
	t := m.tags[m.tagCursor]
	return tagKey{account: t.account, id: t.id}
}

func (m *model) currentMessages() []provider.Message {
	return m.messages[m.currentTagKey()]
}

func (m *model) syncPreview() {
	msgs := m.currentMessages()
	m.preview.SetContent("")
	if m.listCursor < len(msgs) {
		msg := msgs[m.listCursor]
		header := fmt.Sprintf("From: %s\nSubject: %s\nDate: %s\n\n",
			msg.From, msg.Subject, msg.Date.Format("2006-01-02 15:04"))
		body := renderMessageBody(m.theme, m.preview.Width(), msg)
		m.preview.SetContent(header + body)
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
