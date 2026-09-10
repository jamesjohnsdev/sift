package outlook

import (
	"context"
	"errors"
	"io"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

// ErrNotImplemented marks provider.Provider methods this package doesn't
// implement yet - sending and attachment download land in a follow-up.
var ErrNotImplemented = errors.New("outlook: not implemented yet")

func (p *Provider) Attachment(ctx context.Context, msg provider.MessageID, att provider.AttachmentID) (io.ReadCloser, error) {
	return nil, ErrNotImplemented
}

func (p *Provider) Send(ctx context.Context, draft provider.Draft) error {
	return ErrNotImplemented
}
