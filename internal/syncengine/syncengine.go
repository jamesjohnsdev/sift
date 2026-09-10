// Package syncengine pulls mail from each configured account's
// provider.Provider into the local storage.Store, and keeps it updated
// via Watch for as long as the app is running.
package syncengine

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/oauth2"

	"github.com/jamesjohnsdev/sift/internal/auth"
	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/provider"
	"github.com/jamesjohnsdev/sift/internal/provider/gmail"
	"github.com/jamesjohnsdev/sift/internal/provider/outlook"
	"github.com/jamesjohnsdev/sift/internal/storage"
)

// ProviderFactory builds the provider.Provider for one account. Tests
// substitute a fake; production uses DefaultProviderFactory.
type ProviderFactory func(ctx context.Context, acc config.Account, tok *oauth2.Token) (provider.Provider, error)

// DefaultProviderFactory builds the real Gmail/Outlook client for acc.
func DefaultProviderFactory(ctx context.Context, acc config.Account, tok *oauth2.Token) (provider.Provider, error) {
	switch acc.Kind {
	case "gmail":
		return gmail.New(ctx, provider.AccountID(acc.ID), acc.ClientID, tok), nil
	case "outlook":
		return outlook.New(ctx, provider.AccountID(acc.ID), acc.ClientID, tok), nil
	default:
		return nil, fmt.Errorf("account %s: unknown kind %q", acc.ID, acc.Kind)
	}
}

type Manager struct {
	store       *storage.Store
	accounts    []config.Account
	newProvider ProviderFactory
	updates     chan struct{}
}

// New builds a Manager. A nil newProvider defaults to
// DefaultProviderFactory; tests pass a fake instead.
func New(store *storage.Store, accounts []config.Account, newProvider ProviderFactory) *Manager {
	if newProvider == nil {
		newProvider = DefaultProviderFactory
	}
	return &Manager{store: store, accounts: accounts, newProvider: newProvider, updates: make(chan struct{}, 1)}
}

// Updates receives a pulse whenever Watch writes new data to storage, so a
// long-running consumer (the TUI) knows to reload. It's buffered and
// coalescing: any number of writes between reads collapse into one pulse,
// so a slow consumer never blocks Watch and never needs more than "reload,
// something changed" - not what changed.
func (m *Manager) Updates() <-chan struct{} {
	return m.updates
}

func (m *Manager) signalUpdate() {
	select {
	case m.updates <- struct{}{}:
	default:
	}
}

// authenticatedProvider loads acc's token and builds its provider - the
// shared first step of both syncing and sending.
func (m *Manager) authenticatedProvider(ctx context.Context, acc config.Account) (provider.Provider, error) {
	tok, err := auth.LoadToken(acc.ID)
	if err != nil {
		return nil, fmt.Errorf("account %s: load token: %w", acc.ID, err)
	}
	if tok == nil {
		return nil, fmt.Errorf("account %s: not authenticated yet", acc.ID)
	}

	p, err := m.newProvider(ctx, acc, tok)
	if err != nil {
		return nil, fmt.Errorf("account %s: build provider: %w", acc.ID, err)
	}
	return p, nil
}

func (m *Manager) accountByID(id provider.AccountID) (config.Account, bool) {
	for _, acc := range m.accounts {
		if acc.ID == string(id) {
			return acc, true
		}
	}
	return config.Account{}, false
}

// Send authenticates accountID's provider and sends draft through it.
// Matches tui.SendFunc's signature so it can be passed straight through.
func (m *Manager) Send(ctx context.Context, accountID provider.AccountID, draft provider.Draft) error {
	acc, ok := m.accountByID(accountID)
	if !ok {
		return fmt.Errorf("account %s: not configured", accountID)
	}

	p, err := m.authenticatedProvider(ctx, acc)
	if err != nil {
		return err
	}
	return p.Send(ctx, draft)
}

// SyncAccount loads acc's token, builds its provider, and pulls the
// account's tags plus the first page of each tag's messages into
// storage. It does not paginate or backfill further history - that's a
// separate, later increment.
func (m *Manager) SyncAccount(ctx context.Context, acc config.Account) error {
	_, err := m.syncAccount(ctx, acc)
	return err
}

// syncAccount is SyncAccount's implementation, additionally returning the
// built provider so Start can hand it straight to watch without
// re-authenticating and re-constructing it.
func (m *Manager) syncAccount(ctx context.Context, acc config.Account) (provider.Provider, error) {
	p, err := m.authenticatedProvider(ctx, acc)
	if err != nil {
		return nil, err
	}

	accountID := provider.AccountID(acc.ID)
	if err := m.store.UpsertAccount(ctx, storage.Account{ID: accountID, Kind: p.Kind(), Email: acc.Email}); err != nil {
		return nil, fmt.Errorf("account %s: upsert account: %w", acc.ID, err)
	}

	tags, err := p.Tags(ctx)
	if err != nil {
		return nil, fmt.Errorf("account %s: fetch tags: %w", acc.ID, err)
	}
	if err := m.store.UpsertTags(ctx, accountID, tags); err != nil {
		return nil, fmt.Errorf("account %s: upsert tags: %w", acc.ID, err)
	}

	for _, tag := range tags {
		page, err := p.Messages(ctx, tag.ID, "")
		if err != nil {
			return nil, fmt.Errorf("account %s: fetch messages for tag %s: %w", acc.ID, tag.ID, err)
		}
		if err := m.store.UpsertMessages(ctx, accountID, page.Messages); err != nil {
			return nil, fmt.Errorf("account %s: upsert messages for tag %s: %w", acc.ID, tag.ID, err)
		}
	}

	return p, nil
}

// Start runs SyncAccount for every configured account concurrently - a
// slow or broken account must not block the others - then launches a
// background Watch loop per successfully synced account. Start returns
// once every account's initial sync attempt has finished; the Watch
// goroutines keep running until ctx is done.
func (m *Manager) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, acc := range m.accounts {
		wg.Add(1)
		go func(acc config.Account) {
			defer wg.Done()

			p, err := m.syncAccount(ctx, acc)
			if err != nil {
				fmt.Fprintln(os.Stderr, "sift:", err)
				return
			}
			go m.watch(ctx, acc, p)
		}(acc)
	}
	wg.Wait()
}

func (m *Manager) watch(ctx context.Context, acc config.Account, p provider.Provider) {
	updates, err := p.Watch(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sift: account", acc.ID, "watch:", err)
		return
	}

	accountID := provider.AccountID(acc.ID)
	for update := range updates {
		if update.Kind != provider.MessageAdded {
			continue
		}
		msg, err := p.Message(ctx, update.MessageID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sift: account", acc.ID, "fetch message:", err)
			continue
		}
		if err := m.store.UpsertMessages(ctx, accountID, []provider.Message{*msg}); err != nil {
			fmt.Fprintln(os.Stderr, "sift: account", acc.ID, "upsert message:", err)
			continue
		}
		m.signalUpdate()
	}
}
