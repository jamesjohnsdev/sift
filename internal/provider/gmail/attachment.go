package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

type attachmentResponse struct {
	Size int    `json:"size"`
	Data string `json:"data"`
}

func (p *Provider) Attachment(ctx context.Context, msg provider.MessageID, att provider.AttachmentID) (io.ReadCloser, error) {
	path := "/messages/" + url.PathEscape(string(msg)) + "/attachments/" + url.PathEscape(string(att))

	var resp attachmentResponse
	if err := p.getJSON(ctx, path, &resp); err != nil {
		return nil, err
	}

	data, err := base64.RawURLEncoding.DecodeString(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("decode attachment data: %w", err)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
