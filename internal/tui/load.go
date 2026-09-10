package tui

import (
	"context"
	"sort"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
	"github.com/jamesjohnsdev/sift/internal/storage"
)

// messagePageSize matches the page-size convention used by the Gmail and
// Outlook provider implementations.
const messagePageSize = 50

type tag struct {
	account provider.AccountID
	id      provider.TagID
	name    string
}

type tagKey struct {
	account provider.AccountID
	id      provider.TagID
}

// specialOrder puts the tags a mail client user expects up top; anything
// not listed here (provider.TagNone, i.e. every freeform tag) sorts after,
// alphabetically.
var specialOrder = map[provider.SpecialTag]int{
	provider.TagInbox:  0,
	provider.TagSent:   1,
	provider.TagDrafts: 2,
	provider.TagTrash:  3,
	provider.TagSpam:   4,
}

// loadData reads every configured account's tags and messages from store.
// Tag names are prefixed with the owning account's email only when more
// than one account is configured, since a single flat list otherwise
// can't tell two accounts' tags apart.
func loadData(ctx context.Context, store *storage.Store) ([]tag, map[tagKey][]provider.Message, error) {
	accounts, err := store.Accounts(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(accounts) == 0 {
		return []tag{{name: "(no accounts configured)"}}, map[tagKey][]provider.Message{}, nil
	}
	multiAccount := len(accounts) > 1

	var tags []tag
	messages := make(map[tagKey][]provider.Message)

	for _, acc := range accounts {
		accTags, err := store.Tags(ctx, acc.ID)
		if err != nil {
			return nil, nil, err
		}
		sortTags(accTags)

		if len(accTags) == 0 {
			tags = append(tags, tag{account: acc.ID, name: displayName(acc.Email, "(no tags yet)", multiAccount)})
			continue
		}

		for _, pt := range accTags {
			tags = append(tags, tag{
				account: acc.ID,
				id:      pt.ID,
				name:    displayName(acc.Email, pt.Name, multiAccount),
			})

			msgs, err := store.Messages(ctx, acc.ID, pt.ID, messagePageSize, time.Time{})
			if err != nil {
				return nil, nil, err
			}
			messages[tagKey{account: acc.ID, id: pt.ID}] = msgs
		}
	}

	return tags, messages, nil
}

func displayName(email, name string, multiAccount bool) string {
	if !multiAccount {
		return name
	}
	return email + ": " + name
}

func sortTags(tags []provider.Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		oi, iSpecial := specialOrder[tags[i].Special]
		oj, jSpecial := specialOrder[tags[j].Special]
		if iSpecial && jSpecial {
			return oi < oj
		}
		if iSpecial != jSpecial {
			return iSpecial
		}
		return tags[i].Name < tags[j].Name
	})
}
