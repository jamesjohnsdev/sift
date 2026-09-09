package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "sift.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

func TestAccountsRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	want := Account{ID: "acct-1", Kind: provider.Gmail, Email: "me@example.com"}
	if err := s.UpsertAccount(ctx, want); err != nil {
		t.Fatalf("UpsertAccount: %v", err)
	}

	got, err := s.Accounts(ctx)
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("Accounts = %+v, want [%+v]", got, want)
	}
}

func TestTagsRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	account := seedAccount(t, s)

	tags := []provider.Tag{
		{ID: "inbox", Name: "Inbox", Special: provider.TagInbox},
		{ID: "work/projects", Name: "Work/Projects"},
	}
	if err := s.UpsertTags(ctx, account, tags); err != nil {
		t.Fatalf("UpsertTags: %v", err)
	}

	got, err := s.Tags(ctx, account)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Tags = %+v, want 2 entries", got)
	}
}

func TestMessagesUpsertListGetSearch(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	account := seedAccount(t, s)

	if err := s.UpsertTags(ctx, account, []provider.Tag{{ID: "inbox", Name: "Inbox", Special: provider.TagInbox}}); err != nil {
		t.Fatalf("UpsertTags: %v", err)
	}

	msg := provider.Message{
		ID:       "msg-1",
		ThreadID: "thread-1",
		From:     "alice@example.com",
		To:       []string{"me@example.com"},
		Subject:  "Q3 roadmap review",
		Date:     time.Unix(1700000000, 0),
		Tags:     []provider.TagID{"inbox"},
		Snippet:  "Can we push the review...",
		BodyText: "Can we push the roadmap review to Thursday?",
		Attachments: []provider.Attachment{
			{ID: "att-1", Filename: "roadmap.pdf", MIMEType: "application/pdf", Size: 1024},
		},
	}
	if err := s.UpsertMessages(ctx, account, []provider.Message{msg}); err != nil {
		t.Fatalf("UpsertMessages: %v", err)
	}

	list, err := s.Messages(ctx, account, "inbox", 10, time.Time{})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(list) != 1 || list[0].ID != msg.ID {
		t.Fatalf("Messages = %+v, want [%s]", list, msg.ID)
	}
	if len(list[0].Tags) != 1 || list[0].Tags[0] != "inbox" {
		t.Fatalf("Messages[0].Tags = %v, want [inbox]", list[0].Tags)
	}

	got, err := s.Message(ctx, account, msg.ID)
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if got == nil || got.Subject != msg.Subject {
		t.Fatalf("Message = %+v, want subject %q", got, msg.Subject)
	}
	if len(got.Attachments) != 1 || got.Attachments[0].Filename != "roadmap.pdf" {
		t.Fatalf("Message.Attachments = %+v", got.Attachments)
	}

	missing, err := s.Message(ctx, account, "does-not-exist")
	if err != nil {
		t.Fatalf("Message(missing): %v", err)
	}
	if missing != nil {
		t.Fatalf("Message(missing) = %+v, want nil", missing)
	}

	found, err := s.Search(ctx, account, "roadmap", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(found) != 1 || found[0].ID != msg.ID {
		t.Fatalf("Search = %+v, want [%s]", found, msg.ID)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	account := seedAccount(t, s)

	if got, err := s.Cursor(ctx, account); err != nil || got != "" {
		t.Fatalf("Cursor(unset) = %q, %v, want empty, nil", got, err)
	}

	if err := s.SetCursor(ctx, account, "history-42"); err != nil {
		t.Fatalf("SetCursor: %v", err)
	}
	got, err := s.Cursor(ctx, account)
	if err != nil {
		t.Fatalf("Cursor: %v", err)
	}
	if got != "history-42" {
		t.Fatalf("Cursor = %q, want history-42", got)
	}
}

func TestOutboxLifecycle(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	account := seedAccount(t, s)

	id, err := s.Enqueue(ctx, account, provider.Draft{
		To:           []string{"bob@example.com"},
		Subject:      "Hello",
		MarkdownBody: "**hi**",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	pending, err := s.Pending(ctx, account)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != id || pending[0].Draft.Subject != "Hello" {
		t.Fatalf("Pending = %+v", pending)
	}

	if err := s.MarkFailed(ctx, id, errors.New("smtp timeout"), 3); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	pending, err = s.Pending(ctx, account)
	if err != nil {
		t.Fatalf("Pending after 1 failure: %v", err)
	}
	if len(pending) != 1 || pending[0].Attempts != 1 {
		t.Fatalf("Pending after 1 failure = %+v, want attempts=1", pending)
	}

	for range 2 {
		if err := s.MarkFailed(ctx, id, errors.New("smtp timeout"), 3); err != nil {
			t.Fatalf("MarkFailed: %v", err)
		}
	}
	pending, err = s.Pending(ctx, account)
	if err != nil {
		t.Fatalf("Pending after max retries: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("Pending after max retries = %+v, want empty (surfaced as failed)", pending)
	}

	id2, err := s.Enqueue(ctx, account, provider.Draft{To: []string{"carol@example.com"}, Subject: "Bye"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := s.MarkSent(ctx, id2); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}
	pending, err = s.Pending(ctx, account)
	if err != nil {
		t.Fatalf("Pending after sent: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("Pending after sent = %+v, want empty", pending)
	}
}

func seedAccount(t *testing.T, s *Store) provider.AccountID {
	t.Helper()
	account := Account{ID: "acct-1", Kind: provider.Gmail, Email: "me@example.com"}
	if err := s.UpsertAccount(context.Background(), account); err != nil {
		t.Fatalf("seedAccount: %v", err)
	}
	return account.ID
}
