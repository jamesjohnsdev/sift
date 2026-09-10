package tui

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
	"github.com/jamesjohnsdev/sift/internal/storage"
)

func openTestStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.Open(filepath.Join(t.TempDir(), "sift.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

func TestLoadDataNoAccounts(t *testing.T) {
	store := openTestStore(t)

	tags, messages, err := loadData(context.Background(), store)
	if err != nil {
		t.Fatalf("loadData: %v", err)
	}
	if len(tags) != 1 || tags[0].name != "(no accounts configured)" {
		t.Fatalf("tags = %+v, want single (no accounts configured) entry", tags)
	}
	if len(messages) != 0 {
		t.Fatalf("messages = %+v, want empty", messages)
	}
}

func TestLoadDataAccountWithNoTagsYet(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	if err := store.UpsertAccount(ctx, storage.Account{ID: "acct-1", Kind: provider.Gmail, Email: "me@example.com"}); err != nil {
		t.Fatalf("UpsertAccount: %v", err)
	}

	tags, _, err := loadData(ctx, store)
	if err != nil {
		t.Fatalf("loadData: %v", err)
	}
	if len(tags) != 1 || tags[0].name != "(no tags yet)" {
		t.Fatalf("tags = %+v, want single (no tags yet) entry", tags)
	}
}

func TestLoadDataSingleAccountNoPrefix(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	seedAccountWithInboxAndSent(t, store, "acct-1", "me@example.com")

	tags, messages, err := loadData(ctx, store)
	if err != nil {
		t.Fatalf("loadData: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("tags = %+v, want 2", tags)
	}
	// Special-tag ordering: Inbox before Sent, no email prefix for a single account.
	if tags[0].name != "Inbox" {
		t.Fatalf("tags[0].name = %q, want Inbox", tags[0].name)
	}
	if tags[1].name != "Sent" {
		t.Fatalf("tags[1].name = %q, want Sent", tags[1].name)
	}

	inboxMsgs := messages[tagKey{account: "acct-1", id: "inbox"}]
	if len(inboxMsgs) != 1 || inboxMsgs[0].Subject != "Hello" {
		t.Fatalf("inbox messages = %+v", inboxMsgs)
	}
}

func TestLoadDataMultiAccountPrefixesNames(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	seedAccountWithInboxAndSent(t, store, "acct-1", "me@example.com")
	seedAccountWithInboxAndSent(t, store, "acct-2", "work@example.com")

	tags, _, err := loadData(ctx, store)
	if err != nil {
		t.Fatalf("loadData: %v", err)
	}
	if len(tags) != 4 {
		t.Fatalf("tags = %+v, want 4", tags)
	}
	for _, tg := range tags {
		if tg.name != "me@example.com: Inbox" && tg.name != "me@example.com: Sent" &&
			tg.name != "work@example.com: Inbox" && tg.name != "work@example.com: Sent" {
			t.Fatalf("unexpected tag name %q", tg.name)
		}
	}
}

// seedAccountWithInboxAndSent creates one account with an Inbox (special,
// one message) and a Sent tag (special, no messages), deliberately
// inserted out of the order loadData should sort them into.
func seedAccountWithInboxAndSent(t *testing.T, store *storage.Store, accountID, email string) {
	t.Helper()
	ctx := context.Background()
	account := provider.AccountID(accountID)

	if err := store.UpsertAccount(ctx, storage.Account{ID: account, Kind: provider.Gmail, Email: email}); err != nil {
		t.Fatalf("UpsertAccount: %v", err)
	}
	tags := []provider.Tag{
		{ID: "sent", Name: "Sent", Special: provider.TagSent},
		{ID: "inbox", Name: "Inbox", Special: provider.TagInbox},
	}
	if err := store.UpsertTags(ctx, account, tags); err != nil {
		t.Fatalf("UpsertTags: %v", err)
	}
	msg := provider.Message{
		ID:      provider.MessageID(accountID + "-msg-1"),
		Subject: "Hello",
		From:    "alice@example.com",
		Date:    time.Now(),
		Tags:    []provider.TagID{"inbox"},
	}
	if err := store.UpsertMessages(ctx, account, []provider.Message{msg}); err != nil {
		t.Fatalf("UpsertMessages: %v", err)
	}
}
