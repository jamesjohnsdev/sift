package gmail

import (
	"context"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

var specialLabels = map[string]provider.SpecialTag{
	"INBOX": provider.TagInbox,
	"SENT":  provider.TagSent,
	"DRAFT": provider.TagDrafts,
	"TRASH": provider.TagTrash,
	"SPAM":  provider.TagSpam,
}

type labelsResponse struct {
	Labels []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"labels"`
}

// Tags surfaces the handful of special system labels plus every
// user-defined label; the noisier system labels (CATEGORY_*, UNREAD,
// STARRED, IMPORTANT, CHAT, ...) are left out of the tag list.
func (p *Provider) Tags(ctx context.Context) ([]provider.Tag, error) {
	var resp labelsResponse
	if err := p.getJSON(ctx, "/labels", &resp); err != nil {
		return nil, err
	}

	var tags []provider.Tag
	for _, l := range resp.Labels {
		special, isSpecial := specialLabels[l.ID]
		if l.Type != "user" && !isSpecial {
			continue
		}
		tags = append(tags, provider.Tag{
			ID:      provider.TagID(l.ID),
			Name:    l.Name,
			Special: special,
		})
	}
	return tags, nil
}
