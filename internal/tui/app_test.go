package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/provider"
)

// TestWindowSizeMsgBeforeComposeOpenedDoesNotPanic guards against a real
// crash: WindowSizeMsg used to unconditionally resize m.compose, but
// m.compose is only actually constructed (via newComposeModel/
// newReplyModel) once compose is opened - resizing the zero-value
// composeModel before that panicked inside textarea's SetWidth on every
// single startup, since WindowSizeMsg fires before any key is ever
// pressed.
func TestWindowSizeMsgBeforeComposeOpenedDoesNotPanic(t *testing.T) {
	store := openTestStore(t)
	m, err := newModel(config.Default(), store, nil, nil)
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Update(WindowSizeMsg) panicked before compose was ever opened: %v", r)
		}
	}()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m2 == nil {
		t.Fatal("Update returned a nil model")
	}
}

func TestReloadPicksUpNewDataAndPreservesSelection(t *testing.T) {
	store := openTestStore(t)
	seedAccountWithInboxAndSent(t, store, "acct-1", "me@example.com")

	m, err := newModel(config.Default(), store, nil, nil)
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}

	// Select Sent (index 1: Inbox sorts first) before reloading, to prove
	// reload doesn't just snap back to the top.
	m.tagCursor = 1
	if got := m.currentTagName(); got != "Sent" {
		t.Fatalf("currentTagName() = %q, want Sent (test setup assumption)", got)
	}

	if err := store.UpsertMessages(context.Background(), "acct-1", []provider.Message{
		{ID: "new-msg", Subject: "Fresh from Watch", Tags: []provider.TagID{"sent"}},
	}); err != nil {
		t.Fatalf("UpsertMessages: %v", err)
	}

	m.reload()

	if got := m.currentTagName(); got != "Sent" {
		t.Fatalf("after reload currentTagName() = %q, want Sent (selection should survive)", got)
	}
	msgs := m.currentMessages()
	if len(msgs) != 1 || msgs[0].Subject != "Fresh from Watch" {
		t.Fatalf("after reload currentMessages() = %+v, want the newly upserted message", msgs)
	}
}

func TestListenForUpdatesTurnsPulseIntoRefreshMsg(t *testing.T) {
	updates := make(chan struct{}, 1)
	updates <- struct{}{}

	cmd := listenForUpdates(updates)
	if cmd == nil {
		t.Fatal("listenForUpdates(non-nil channel) = nil, want a command")
	}
	if _, ok := cmd().(refreshMsg); !ok {
		t.Fatalf("listenForUpdates command returned %#v, want refreshMsg", cmd())
	}
}

func TestListenForUpdatesNilChannel(t *testing.T) {
	if cmd := listenForUpdates(nil); cmd != nil {
		t.Fatal("listenForUpdates(nil) should return a nil command, not one that blocks forever")
	}
}
