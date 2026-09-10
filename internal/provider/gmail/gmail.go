// Package gmail implements provider.Provider against the Gmail REST API.
package gmail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/jamesjohnsdev/sift/internal/auth"
	"github.com/jamesjohnsdev/sift/internal/provider"
)

const defaultBaseURL = "https://gmail.googleapis.com/gmail/v1/users/me"

type Provider struct {
	account provider.AccountID
	client  *http.Client
	baseURL string
}

var _ provider.Provider = (*Provider)(nil)

// New builds a Gmail provider for account, authenticating with tok and
// persisting it to the OS keychain (keyed by account) whenever it's
// refreshed.
func New(ctx context.Context, account provider.AccountID, clientID string, tok *oauth2.Token) *Provider {
	oconf := &oauth2.Config{ClientID: clientID, Endpoint: auth.Gmail(clientID).Endpoint}
	client := provider.NewHTTPClient(ctx, oconf, tok, func(refreshed *oauth2.Token) {
		_ = auth.SaveToken(string(account), refreshed)
	})
	return &Provider{account: account, client: client, baseURL: defaultBaseURL}
}

func (p *Provider) Kind() provider.Kind         { return provider.Gmail }
func (p *Provider) Account() provider.AccountID { return p.account }

// getJSON issues a GET against the Gmail API and decodes the JSON body
// into out. path is relative to baseURL unless it's already an absolute
// URL (as Gmail's own pageToken-driven requests always are here).
func (p *Provider) getJSON(ctx context.Context, path string, out any) (err error) {
	url := path
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = p.baseURL + path
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("gmail request: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("gmail request to %s: status %d: %s", path, resp.StatusCode, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode gmail response: %w", err)
	}
	return nil
}

// postJSON issues a POST against the Gmail API with a JSON-encoded body
// and decodes the JSON response into out (which may be nil to discard it).
func (p *Provider) postJSON(ctx context.Context, path string, body, out any) (err error) {
	url := path
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = p.baseURL + path
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("gmail request: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("gmail request to %s: status %d: %s", path, resp.StatusCode, respBody)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode gmail response: %w", err)
	}
	return nil
}
