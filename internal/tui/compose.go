package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

// SendFunc sends draft on behalf of account. Injected so tui doesn't need
// to know how to authenticate or construct a provider itself.
type SendFunc func(ctx context.Context, account provider.AccountID, draft provider.Draft) error

type composeField int

const (
	fieldTo composeField = iota
	fieldCc
	fieldBcc
	fieldSubject
	fieldBody

	numComposeFields
)

type composeModel struct {
	account   provider.AccountID
	inReplyTo provider.MessageID

	to, cc, bcc, subject textinput.Model
	body                 textarea.Model

	field composeField
	send  SendFunc

	sending bool
	status  string
}

func newComposeModel(send SendFunc, account provider.AccountID) composeModel {
	newField := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		return ti
	}
	body := textarea.New()
	body.Placeholder = "Write your message in Markdown..."

	c := composeModel{
		account: account,
		to:      newField("recipient@example.com"),
		cc:      newField(""),
		bcc:     newField(""),
		subject: newField(""),
		body:    body,
		send:    send,
	}
	c.focusCurrent()
	return c
}

// newReplyModel pre-fills a composeModel to reply to msg.
func newReplyModel(send SendFunc, account provider.AccountID, msg provider.Message) composeModel {
	c := newComposeModel(send, account)
	c.inReplyTo = msg.ID
	c.to.SetValue(msg.From)

	subject := msg.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	c.subject.SetValue(subject)
	c.body.SetValue(quoteReply(msg))
	return c
}

func quoteReply(msg provider.Message) string {
	body := msg.BodyText
	if body == "" {
		body = msg.BodyHTML
	}
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		lines[i] = "> " + l
	}
	return "\n\n" + msg.From + " wrote:\n" + strings.Join(lines, "\n")
}

func (c *composeModel) applySize(width, height int) {
	fieldWidth := max(width-20, 10)
	c.to.SetWidth(fieldWidth)
	c.cc.SetWidth(fieldWidth)
	c.bcc.SetWidth(fieldWidth)
	c.subject.SetWidth(fieldWidth)
	c.body.SetWidth(max(width-4, 10))
	c.body.SetHeight(max(height-12, 3))
}

func (c *composeModel) focusCurrent() {
	c.to.Blur()
	c.cc.Blur()
	c.bcc.Blur()
	c.subject.Blur()
	c.body.Blur()
	switch c.field {
	case fieldTo:
		c.to.Focus()
	case fieldCc:
		c.cc.Focus()
	case fieldBcc:
		c.bcc.Focus()
	case fieldSubject:
		c.subject.Focus()
	case fieldBody:
		c.body.Focus()
	}
}

func (c composeModel) updateField(msg tea.KeyPressMsg) (composeModel, tea.Cmd) {
	var cmd tea.Cmd
	switch c.field {
	case fieldTo:
		c.to, cmd = c.to.Update(msg)
	case fieldCc:
		c.cc, cmd = c.cc.Update(msg)
	case fieldBcc:
		c.bcc, cmd = c.bcc.Update(msg)
	case fieldSubject:
		c.subject, cmd = c.subject.Update(msg)
	case fieldBody:
		c.body, cmd = c.body.Update(msg)
	}
	return c, cmd
}

// composeSentMsg reports the result of a background Send.
type composeSentMsg struct{ err error }

func (c composeModel) submit() tea.Cmd {
	draft := provider.Draft{
		To:           splitAddresses(c.to.Value()),
		Cc:           splitAddresses(c.cc.Value()),
		Bcc:          splitAddresses(c.bcc.Value()),
		Subject:      c.subject.Value(),
		MarkdownBody: c.body.Value(),
		InReplyTo:    c.inReplyTo,
	}
	account := c.account
	send := c.send
	return func() tea.Msg {
		return composeSentMsg{err: send(context.Background(), account, draft)}
	}
}

func splitAddresses(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
