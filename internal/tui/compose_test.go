package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

func TestSplitAddresses(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"", nil},
		{"a@x.com", []string{"a@x.com"}},
		{"a@x.com, b@x.com", []string{"a@x.com", "b@x.com"}},
		{" a@x.com ,, b@x.com ", []string{"a@x.com", "b@x.com"}},
	}
	for _, tt := range tests {
		got := splitAddresses(tt.raw)
		if len(got) != len(tt.want) {
			t.Fatalf("splitAddresses(%q) = %v, want %v", tt.raw, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("splitAddresses(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		}
	}
}

func TestNewReplyModelPrefillsFields(t *testing.T) {
	msg := provider.Message{
		ID:       "m1",
		From:     "alice@example.com",
		Subject:  "Q3 roadmap",
		Date:     time.Now(),
		BodyText: "line one\nline two",
	}

	c := newReplyModel(nil, "acct-1", msg)

	if c.to.Value() != "alice@example.com" {
		t.Fatalf("To = %q, want alice@example.com", c.to.Value())
	}
	if c.subject.Value() != "Re: Q3 roadmap" {
		t.Fatalf("Subject = %q, want %q", c.subject.Value(), "Re: Q3 roadmap")
	}
	if c.inReplyTo != "m1" {
		t.Fatalf("inReplyTo = %q, want m1", c.inReplyTo)
	}
	if !strings.Contains(c.body.Value(), "> line one") || !strings.Contains(c.body.Value(), "> line two") {
		t.Fatalf("body = %q, want quoted original lines", c.body.Value())
	}
	if c.account != "acct-1" {
		t.Fatalf("account = %q, want acct-1", c.account)
	}
}

func TestNewReplyModelDoesNotDoublePrefixSubject(t *testing.T) {
	msg := provider.Message{Subject: "Re: already replied"}
	c := newReplyModel(nil, "acct-1", msg)
	if c.subject.Value() != "Re: already replied" {
		t.Fatalf("Subject = %q, want unchanged (already has Re:)", c.subject.Value())
	}
}

func TestComposeSubmitCallsSendWithBuiltDraft(t *testing.T) {
	var gotAccount provider.AccountID
	var gotDraft provider.Draft
	send := func(ctx context.Context, account provider.AccountID, draft provider.Draft) error {
		gotAccount = account
		gotDraft = draft
		return nil
	}

	c := newComposeModel(send, "acct-1")
	c.to.SetValue("bob@example.com, carol@example.com")
	c.subject.SetValue("Hello")
	c.body.SetValue("**hi**")

	cmd := c.submit()
	msg, ok := cmd().(composeSentMsg)
	if !ok {
		t.Fatalf("submit() command returned %#v, want composeSentMsg", msg)
	}
	if msg.err != nil {
		t.Fatalf("composeSentMsg.err = %v, want nil", msg.err)
	}

	if gotAccount != "acct-1" {
		t.Fatalf("account passed to send = %q, want acct-1", gotAccount)
	}
	if len(gotDraft.To) != 2 || gotDraft.To[0] != "bob@example.com" || gotDraft.To[1] != "carol@example.com" {
		t.Fatalf("Draft.To = %v", gotDraft.To)
	}
	if gotDraft.Subject != "Hello" || gotDraft.MarkdownBody != "**hi**" {
		t.Fatalf("Draft = %+v", gotDraft)
	}
}

func TestComposeSubmitPropagatesSendError(t *testing.T) {
	send := func(context.Context, provider.AccountID, provider.Draft) error {
		return errors.New("smtp rejected")
	}
	c := newComposeModel(send, "acct-1")

	msg := c.submit()().(composeSentMsg)
	if msg.err == nil {
		t.Fatal("composeSentMsg.err = nil, want the send error")
	}
}

func TestComposeModelFocusCycling(t *testing.T) {
	c := newComposeModel(nil, "acct-1")
	if !c.to.Focused() {
		t.Fatal("a fresh composeModel should focus the To field")
	}

	c.field = fieldBody
	c.focusCurrent()
	if !c.body.Focused() || c.to.Focused() {
		t.Fatal("focusCurrent should focus only the current field")
	}
}
