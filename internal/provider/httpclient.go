package provider

import (
	"context"
	"net/http"
	"sync"

	"golang.org/x/oauth2"
)

// NewHTTPClient returns an http.Client that authenticates requests with
// tok, transparently refreshing it via oconf as needed. onRefresh, if
// non-nil, is called whenever the token actually changes - e.g. to
// persist the new refresh token to the OS keychain.
func NewHTTPClient(ctx context.Context, oconf *oauth2.Config, tok *oauth2.Token, onRefresh func(*oauth2.Token)) *http.Client {
	src := &notifyingTokenSource{
		base:      oconf.TokenSource(ctx, tok),
		onRefresh: onRefresh,
		last:      tok,
	}
	return oauth2.NewClient(ctx, src)
}

type notifyingTokenSource struct {
	base      oauth2.TokenSource
	onRefresh func(*oauth2.Token)

	mu   sync.Mutex
	last *oauth2.Token
}

func (s *notifyingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	changed := s.last == nil || tok.AccessToken != s.last.AccessToken
	s.last = tok
	s.mu.Unlock()

	if changed && s.onRefresh != nil {
		s.onRefresh(tok)
	}
	return tok, nil
}
