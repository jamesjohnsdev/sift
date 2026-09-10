package outlook

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

type attachmentContentResponse struct {
	ContentBytes string `json:"contentBytes"`
}

// Attachment fetches one attachment's content. Graph's contentBytes is
// standard base64 (not URL-safe, unlike Gmail's attachment data).
func (p *Provider) Attachment(ctx context.Context, msg provider.MessageID, att provider.AttachmentID) (io.ReadCloser, error) {
	var resp attachmentContentResponse
	path := "/messages/" + url.PathEscape(string(msg)) + "/attachments/" + url.PathEscape(string(att))
	if err := p.getJSON(ctx, path, &resp); err != nil {
		return nil, err
	}

	content, err := base64.StdEncoding.DecodeString(resp.ContentBytes)
	if err != nil {
		return nil, fmt.Errorf("decode attachment content: %w", err)
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}
