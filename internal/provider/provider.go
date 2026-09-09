// Package provider is the facade every mail backend implements, so the rest
// of sift never branches on which provider an account uses.
package provider

import (
	"context"
	"io"
)

type Kind string

const (
	Gmail   Kind = "gmail"
	Outlook Kind = "outlook"
)

type Provider interface {
	Kind() Kind
	Account() AccountID

	Tags(ctx context.Context) ([]Tag, error)

	Messages(ctx context.Context, tag TagID, cursor Cursor) (Page, error)
	Message(ctx context.Context, id MessageID) (*Message, error)
	Attachment(ctx context.Context, msg MessageID, att AttachmentID) (io.ReadCloser, error)

	Send(ctx context.Context, draft Draft) error

	// Search runs a provider-native query, used for the `srv:` prefix.
	Search(ctx context.Context, query string) ([]Message, error)

	// Watch streams updates until ctx is done; push where the provider
	// supports it, polling otherwise.
	Watch(ctx context.Context) (<-chan Update, error)
}
