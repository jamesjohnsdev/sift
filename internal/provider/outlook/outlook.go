// Package outlook implements provider.Provider against the Microsoft
// Graph REST API.
package outlook

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

const defaultBaseURL = "https://graph.microsoft.com/v1.0/me"

type Provider struct {
	account provider.AccountID
	client  *http.Client
	baseURL string
}

var _ provider.Provider = (*Provider)(nil)

// New builds an Outlook provider for account, authenticating with tok and
// persisting it to the OS keychain (keyed by account) whenever it's
// refreshed.
func New(ctx context.Context, account provider.AccountID, clientID string, tok *oauth2.Token) *Provider {
	oconf := &oauth2.Config{ClientID: clientID, Endpoint: auth.Outlook(clientID).Endpoint}
	client := provider.NewHTTPClient(ctx, oconf, tok, func(refreshed *oauth2.Token) {
		_ = auth.SaveToken(string(account), refreshed)
	})
	return &Provider{account: account, client: client, baseURL: defaultBaseURL}
}

func (p *Provider) Kind() provider.Kind         { return provider.Outlook }
func (p *Provider) Account() provider.AccountID { return p.account }

// getJSON issues a GET against the Graph API and decodes the JSON body
// into out. path is relative to baseURL unless it's already an absolute
// URL, as an @odata.nextLink page always is.
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
		return fmt.Errorf("graph request: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("graph request to %s: status %d: %s", path, resp.StatusCode, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode graph response: %w", err)
	}
	return nil
}

// postJSON issues a POST against the Graph API with a JSON-encoded body.
// Graph mail actions (like sendMail) return 202 Accepted with an empty
// body on success, so any 2xx status is treated as success and the
// response body is only read for the error path.
func (p *Provider) postJSON(ctx context.Context, path string, body any) (err error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("graph request: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("graph request to %s: status %d: %s", path, resp.StatusCode, respBody)
	}
	return nil
}
