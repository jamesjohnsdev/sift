package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// fakeBrowser simulates the user's browser completing the provider's login
// screen: it reads the redirect_uri and state straight off the auth URL sift
// generated, then hits the callback with the given query additions - it
// never talks to a real authorization server.
func fakeBrowser(t *testing.T, extra url.Values) func(authURL string) error {
	t.Helper()
	return func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		redirect, err := url.Parse(q.Get("redirect_uri"))
		if err != nil {
			return err
		}
		cbq := redirect.Query()
		cbq.Set("state", q.Get("state"))
		for k, v := range extra {
			cbq[k] = v
		}
		redirect.RawQuery = cbq.Encode()

		go func() {
			resp, err := http.Get(redirect.String())
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

func fakeTokenServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "fake-access-token",
			"refresh_token": "fake-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
}

func TestAuthenticateSuccess(t *testing.T) {
	tokenSrv := fakeTokenServer(t)
	defer tokenSrv.Close()

	cfg := ProviderConfig{
		ClientID: "test-client",
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.invalid/authorize", TokenURL: tokenSrv.URL},
		Scopes:   []string{"scope-a"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tok, err := Authenticate(ctx, cfg, fakeBrowser(t, url.Values{"code": {"fake-code"}}))
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if tok.AccessToken != "fake-access-token" {
		t.Fatalf("AccessToken = %q, want fake-access-token", tok.AccessToken)
	}
}

func TestAuthenticateDenied(t *testing.T) {
	cfg := ProviderConfig{
		ClientID: "test-client",
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.invalid/authorize", TokenURL: "https://example.invalid/token"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Authenticate(ctx, cfg, fakeBrowser(t, url.Values{"error": {"access_denied"}}))
	if err == nil {
		t.Fatal("Authenticate: expected error for denied authorization, got nil")
	}
}

func TestAuthenticateStateMismatch(t *testing.T) {
	cfg := ProviderConfig{
		ClientID: "test-client",
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.invalid/authorize", TokenURL: "https://example.invalid/token"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Override state with a wrong value after fakeBrowser sets the real one.
	browser := func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		redirect, err := url.Parse(u.Query().Get("redirect_uri"))
		if err != nil {
			return err
		}
		q := redirect.Query()
		q.Set("state", "wrong-state")
		q.Set("code", "fake-code")
		redirect.RawQuery = q.Encode()

		go func() {
			resp, err := http.Get(redirect.String())
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}

	_, err := Authenticate(ctx, cfg, browser)
	if err == nil {
		t.Fatal("Authenticate: expected error for state mismatch, got nil")
	}
}

func TestAuthenticateContextCanceled(t *testing.T) {
	cfg := ProviderConfig{
		ClientID: "test-client",
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.invalid/authorize", TokenURL: "https://example.invalid/token"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Browser that never actually calls back.
	_, err := Authenticate(ctx, cfg, func(string) error { return nil })
	if err == nil {
		t.Fatal("Authenticate: expected error when nothing calls back before ctx is done, got nil")
	}
}

func TestAuthenticateOpenBrowserError(t *testing.T) {
	cfg := ProviderConfig{
		ClientID: "test-client",
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.invalid/authorize", TokenURL: "https://example.invalid/token"},
	}

	_, err := Authenticate(context.Background(), cfg, func(string) error {
		return context.DeadlineExceeded // any non-nil error stands in here
	})
	if err == nil {
		t.Fatal("Authenticate: expected error when openBrowser fails, got nil")
	}
}
