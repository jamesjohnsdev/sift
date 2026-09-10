package tui

import (
	"context"
	"testing"

	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/provider"
)

func TestReloadPicksUpNewDataAndPreservesSelection(t *testing.T) {
	store := openTestStore(t)
	seedAccountWithInboxAndSent(t, store, "acct-1", "me@example.com")

	m, err := newModel(config.Default(), store, nil)
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
