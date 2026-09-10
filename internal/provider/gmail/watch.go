package gmail

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

const pollInterval = 30 * time.Second

// Watch polls the inbox rather than using real push: Gmail push requires a
// Cloud Pub/Sub topic and a subscribed GCP project, which a desktop app
// has no good way to provision or receive callbacks for.
func (p *Provider) Watch(ctx context.Context) (<-chan provider.Update, error) {
	return provider.PollWatch(ctx, pollInterval, p.inboxMessageIDs), nil
}

func (p *Provider) inboxMessageIDs(ctx context.Context) ([]provider.MessageID, error) {
	q := url.Values{"maxResults": {strconv.Itoa(pageSize)}, "labelIds": {"INBOX"}}

	var list messageListResponse
	if err := p.getJSON(ctx, "/messages?"+q.Encode(), &list); err != nil {
		return nil, err
	}
	ids := make([]provider.MessageID, len(list.Messages))
	for i, m := range list.Messages {
		ids[i] = provider.MessageID(m.ID)
	}
	return ids, nil
}
