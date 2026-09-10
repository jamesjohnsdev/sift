package outlook

import (
	"context"
	"strings"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

var specialFolders = map[string]provider.SpecialTag{
	"inbox":         provider.TagInbox,
	"sent items":    provider.TagSent,
	"drafts":        provider.TagDrafts,
	"deleted items": provider.TagTrash,
	"junk email":    provider.TagSpam,
}

// maxFolderDepth bounds the recursive child-folder walk; real mailboxes
// don't nest anywhere near this deep.
const maxFolderDepth = 5

type folder struct {
	ID               string `json:"id"`
	DisplayName      string `json:"displayName"`
	ChildFolderCount int    `json:"childFolderCount"`
}

type foldersResponse struct {
	Value []folder `json:"value"`
}

// Tags flattens Outlook's folder hierarchy into tags, joining parent and
// child names with "/" (e.g. "Work/Projects") the same way sift renders
// nested Gmail-style tags.
func (p *Provider) Tags(ctx context.Context) ([]provider.Tag, error) {
	return p.listFolders(ctx, "/mailFolders", "", 0)
}

func (p *Provider) listFolders(ctx context.Context, path, parentPath string, depth int) ([]provider.Tag, error) {
	var resp foldersResponse
	if err := p.getJSON(ctx, path+"?$top=100", &resp); err != nil {
		return nil, err
	}

	var tags []provider.Tag
	for _, f := range resp.Value {
		name := f.DisplayName
		if parentPath != "" {
			name = parentPath + "/" + f.DisplayName
		}
		tags = append(tags, provider.Tag{
			ID:      provider.TagID(f.ID),
			Name:    name,
			Special: specialFolders[strings.ToLower(f.DisplayName)],
		})

		if f.ChildFolderCount > 0 && depth < maxFolderDepth {
			children, err := p.listFolders(ctx, "/mailFolders/"+f.ID+"/childFolders", name, depth+1)
			if err != nil {
				return nil, err
			}
			tags = append(tags, children...)
		}
	}
	return tags, nil
}
