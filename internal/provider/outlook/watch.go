package outlook

import (
	"context"
	"net/url"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

const pollInterval = 30 * time.Second

// Watch polls the inbox rather than using real push: Graph push requires a
// publicly reachable HTTPS webhook subscription, which a desktop app has
// no good way to host.
func (p *Provider) Watch(ctx context.Context) (<-chan provider.Update, error) {
	return provider.PollWatch(ctx, pollInterval, p.inboxMessageIDs), nil
}

func (p *Provider) inboxMessageIDs(ctx context.Context) ([]provider.MessageID, error) {
	q := url.Values{"$top": {"50"}, "$select": {"id"}}

	var resp messagesResponse
	if err := p.getJSON(ctx, "/mailFolders/inbox/messages?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	ids := make([]provider.MessageID, len(resp.Value))
	for i, m := range resp.Value {
		ids[i] = provider.MessageID(m.ID)
	}
	return ids, nil
}
