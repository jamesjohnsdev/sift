package syncengine

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"

	"github.com/jamesjohnsdev/sift/internal/auth"
	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/provider"
	"github.com/jamesjohnsdev/sift/internal/storage"
)

// fakeProvider is a minimal provider.Provider test double: fixed tags and
// messages, plus a Watch channel the test controls directly.
type fakeProvider struct {
	account provider.AccountID
	tags    []provider.Tag
	msgs    map[provider.TagID][]provider.Message
	byID    map[provider.MessageID]provider.Message
	updates chan provider.Update

	sendErr    error
	sentDrafts []provider.Draft
}

var _ provider.Provider = (*fakeProvider)(nil)

func (f *fakeProvider) Kind() provider.Kind         { return provider.Gmail }
func (f *fakeProvider) Account() provider.AccountID { return f.account }

func (f *fakeProvider) Tags(ctx context.Context) ([]provider.Tag, error) { return f.tags, nil }

func (f *fakeProvider) Messages(ctx context.Context, tag provider.TagID, cursor provider.Cursor) (provider.Page, error) {
	return provider.Page{Messages: f.msgs[tag]}, nil
}

func (f *fakeProvider) Message(ctx context.Context, id provider.MessageID) (*provider.Message, error) {
	m, ok := f.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &m, nil
}

func (f *fakeProvider) Attachment(ctx context.Context, msg provider.MessageID, att provider.AttachmentID) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeProvider) Send(ctx context.Context, draft provider.Draft) error {
	f.sentDrafts = append(f.sentDrafts, draft)
	return f.sendErr
}

func (f *fakeProvider) Search(ctx context.Context, query string) ([]provider.Message, error) {
	return nil, nil
}

func (f *fakeProvider) Watch(ctx context.Context) (<-chan provider.Update, error) {
	return f.updates, nil
}

func newFakeFactory(byAccount map[string]*fakeProvider) ProviderFactory {
	return func(ctx context.Context, acc config.Account, tok *oauth2.Token) (provider.Provider, error) {
		p, ok := byAccount[acc.ID]
		if !ok {
			return nil, errors.New("no fake provider configured for " + acc.ID)
		}
		return p, nil
	}
}

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

func TestSyncAccountHappyPath(t *testing.T) {
	keyring.MockInit()
	if err := auth.SaveToken("acct-1", &oauth2.Token{AccessToken: "at"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	store := openTestStore(t)
	fp := &fakeProvider{
		account: "acct-1",
		tags:    []provider.Tag{{ID: "inbox", Name: "Inbox", Special: provider.TagInbox}},
		msgs: map[provider.TagID][]provider.Message{
			"inbox": {{ID: "m1", Subject: "Hello", Tags: []provider.TagID{"inbox"}}},
		},
	}
	acc := config.Account{ID: "acct-1", Kind: "gmail", Email: "me@example.com"}

	m := New(store, []config.Account{acc}, newFakeFactory(map[string]*fakeProvider{"acct-1": fp}))
	if err := m.SyncAccount(context.Background(), acc); err != nil {
		t.Fatalf("SyncAccount: %v", err)
	}

	tags, err := store.Tags(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != "inbox" {
		t.Fatalf("Tags = %+v", tags)
	}

	msgs, err := store.Messages(context.Background(), "acct-1", "inbox", 10, time.Time{})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Subject != "Hello" {
		t.Fatalf("Messages = %+v", msgs)
	}
}

func TestSyncAccountNoToken(t *testing.T) {
	keyring.MockInit()
	store := openTestStore(t)
	acc := config.Account{ID: "acct-no-token", Kind: "gmail"}

	m := New(store, []config.Account{acc}, newFakeFactory(nil))
	err := m.SyncAccount(context.Background(), acc)
	if err == nil {
		t.Fatal("SyncAccount: expected error for missing token, got nil")
	}
}

func TestStartSyncsAllAccountsIndependently(t *testing.T) {
	keyring.MockInit()
	if err := auth.SaveToken("good", &oauth2.Token{AccessToken: "at"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	// Deliberately no token saved for "bad".

	store := openTestStore(t)
	goodProvider := &fakeProvider{
		account: "good",
		tags:    []provider.Tag{{ID: "inbox", Name: "Inbox"}},
		msgs: map[provider.TagID][]provider.Message{
			"inbox": {{ID: "m1", Subject: "ok", Tags: []provider.TagID{"inbox"}}},
		},
		updates: make(chan provider.Update),
	}

	accounts := []config.Account{
		{ID: "good", Kind: "gmail"},
		{ID: "bad", Kind: "gmail"},
	}
	m := New(store, accounts, newFakeFactory(map[string]*fakeProvider{"good": goodProvider}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	msgs, err := store.Messages(context.Background(), "good", "inbox", 10, time.Time{})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Messages(good) = %+v, want 1 (bad account's failure must not affect it)", msgs)
	}

	accs, err := store.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(accs) != 1 || accs[0].ID != "good" {
		t.Fatalf("Accounts = %+v, want only 'good' (bad never got far enough to upsert)", accs)
	}

	close(goodProvider.updates)
}

func TestWatchUpsertsNewMessage(t *testing.T) {
	keyring.MockInit()
	if err := auth.SaveToken("acct-1", &oauth2.Token{AccessToken: "at"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	store := openTestStore(t)
	fp := &fakeProvider{
		account: "acct-1",
		tags:    []provider.Tag{{ID: "inbox", Name: "Inbox"}},
		msgs:    map[provider.TagID][]provider.Message{"inbox": {}},
		byID: map[provider.MessageID]provider.Message{
			"new-1": {ID: "new-1", Subject: "Fresh", Tags: []provider.TagID{"inbox"}},
		},
		updates: make(chan provider.Update, 1),
	}
	acc := config.Account{ID: "acct-1", Kind: "gmail"}
	m := New(store, []config.Account{acc}, newFakeFactory(map[string]*fakeProvider{"acct-1": fp}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	fp.updates <- provider.Update{Kind: provider.MessageAdded, MessageID: "new-1"}

	deadline := time.Now().Add(2 * time.Second)
	for {
		msgs, err := store.Messages(context.Background(), "acct-1", "inbox", 10, time.Time{})
		if err != nil {
			t.Fatalf("Messages: %v", err)
		}
		if len(msgs) == 1 && msgs[0].ID == "new-1" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Messages = %+v, want the watched message to appear within the deadline", msgs)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWatchSignalsUpdates(t *testing.T) {
	keyring.MockInit()
	if err := auth.SaveToken("acct-1", &oauth2.Token{AccessToken: "at"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	store := openTestStore(t)
	fp := &fakeProvider{
		account: "acct-1",
		tags:    []provider.Tag{{ID: "inbox", Name: "Inbox"}},
		msgs:    map[provider.TagID][]provider.Message{"inbox": {}},
		byID: map[provider.MessageID]provider.Message{
			"new-1": {ID: "new-1", Subject: "Fresh", Tags: []provider.TagID{"inbox"}},
			"new-2": {ID: "new-2", Subject: "Fresher", Tags: []provider.TagID{"inbox"}},
		},
		updates: make(chan provider.Update, 2),
	}
	acc := config.Account{ID: "acct-1", Kind: "gmail"}
	m := New(store, []config.Account{acc}, newFakeFactory(map[string]*fakeProvider{"acct-1": fp}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	select {
	case <-m.Updates():
		t.Fatal("Updates fired before any Watch update was sent")
	default:
	}

	// Two updates must still coalesce into one pending pulse, not queue up
	// extra pulses. Wait for both to actually land in storage first, so
	// both signalUpdate calls have definitely already happened - otherwise
	// draining the first pulse as soon as it appears can race ahead of the
	// second message's upsert+signal, making them look uncoalesced.
	fp.updates <- provider.Update{Kind: provider.MessageAdded, MessageID: "new-1"}
	fp.updates <- provider.Update{Kind: provider.MessageAdded, MessageID: "new-2"}

	deadline := time.Now().Add(2 * time.Second)
	for {
		msgs, err := store.Messages(context.Background(), "acct-1", "inbox", 10, time.Time{})
		if err != nil {
			t.Fatalf("Messages: %v", err)
		}
		if len(msgs) == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Messages = %+v, want both watched messages landed before checking coalescing", msgs)
		}
		time.Sleep(10 * time.Millisecond)
	}

	select {
	case <-m.Updates():
	default:
		t.Fatal("Updates: expected a pending pulse after Watch upserted messages")
	}

	select {
	case <-m.Updates():
		t.Fatal("Updates: expected the two updates to coalesce into one pulse, got a second one")
	default:
	}
}

func TestManagerSend(t *testing.T) {
	keyring.MockInit()
	if err := auth.SaveToken("acct-1", &oauth2.Token{AccessToken: "at"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	store := openTestStore(t)
	fp := &fakeProvider{account: "acct-1"}
	acc := config.Account{ID: "acct-1", Kind: "gmail"}
	m := New(store, []config.Account{acc}, newFakeFactory(map[string]*fakeProvider{"acct-1": fp}))

	draft := provider.Draft{To: []string{"bob@example.com"}, Subject: "hi"}
	if err := m.Send(context.Background(), "acct-1", draft); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(fp.sentDrafts) != 1 || fp.sentDrafts[0].Subject != "hi" {
		t.Fatalf("sentDrafts = %+v, want [%+v]", fp.sentDrafts, draft)
	}
}

func TestManagerSendUnknownAccount(t *testing.T) {
	keyring.MockInit()
	store := openTestStore(t)
	m := New(store, nil, newFakeFactory(nil))

	if err := m.Send(context.Background(), "nope", provider.Draft{}); err == nil {
		t.Fatal("Send: expected error for an account not in config, got nil")
	}
}

func TestManagerSendNoToken(t *testing.T) {
	keyring.MockInit()
	store := openTestStore(t)
	acc := config.Account{ID: "acct-no-token", Kind: "gmail"}
	m := New(store, []config.Account{acc}, newFakeFactory(nil))

	if err := m.Send(context.Background(), "acct-no-token", provider.Draft{}); err == nil {
		t.Fatal("Send: expected error when there's no token yet, got nil")
	}
}

func TestManagerSendProviderError(t *testing.T) {
	keyring.MockInit()
	if err := auth.SaveToken("acct-1", &oauth2.Token{AccessToken: "at"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	store := openTestStore(t)
	fp := &fakeProvider{account: "acct-1", sendErr: errors.New("smtp rejected")}
	acc := config.Account{ID: "acct-1", Kind: "gmail"}
	m := New(store, []config.Account{acc}, newFakeFactory(map[string]*fakeProvider{"acct-1": fp}))

	if err := m.Send(context.Background(), "acct-1", provider.Draft{}); err == nil {
		t.Fatal("Send: expected the provider's error to propagate, got nil")
	}
}
