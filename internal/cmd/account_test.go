package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"

	"github.com/jamesjohnsdev/sift/internal/auth"
	"github.com/jamesjohnsdev/sift/internal/config"
)

func fakeAuthenticate(tok *oauth2.Token, err error) authenticateFunc {
	return func(context.Context, auth.ProviderConfig, func(string) error) (*oauth2.Token, error) {
		return tok, err
	}
}

func neverOpenBrowser(string) error {
	return errors.New("should not be called: authenticate is faked")
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	return buf.String()
}

func TestAddAccountWritesFreshInitLua(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()

	err := addAccount("gmail", "me@example.com", "client-abc", "", dir,
		fakeAuthenticate(&oauth2.Token{AccessToken: "at"}, nil), neverOpenBrowser)
	if err != nil {
		t.Fatalf("addAccount: %v", err)
	}

	tok, err := auth.LoadToken("me@example.com")
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if tok == nil || tok.AccessToken != "at" {
		t.Fatalf("LoadToken = %+v, want the authenticated token saved under the default id (email)", tok)
	}

	cfg, err := config.Load(filepath.Join(dir, "init.lua"))
	if err != nil {
		t.Fatalf("config.Load(written init.lua): %v", err)
	}
	if len(cfg.Accounts) != 1 {
		t.Fatalf("Accounts = %+v, want 1", cfg.Accounts)
	}
	want := config.Account{ID: "me@example.com", Kind: "gmail", Email: "me@example.com", ClientID: "client-abc"}
	if cfg.Accounts[0] != want {
		t.Fatalf("Accounts[0] = %+v, want %+v", cfg.Accounts[0], want)
	}
}

func TestAddAccountUsesExplicitID(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()

	err := addAccount("outlook", "me@example.com", "client-abc", "work", dir,
		fakeAuthenticate(&oauth2.Token{AccessToken: "at"}, nil), neverOpenBrowser)
	if err != nil {
		t.Fatalf("addAccount: %v", err)
	}

	if tok, _ := auth.LoadToken("me@example.com"); tok != nil {
		t.Fatal("LoadToken(email) should be empty when an explicit id was given")
	}
	tok, err := auth.LoadToken("work")
	if err != nil || tok == nil {
		t.Fatalf("LoadToken(work) = %v, %v, want the saved token", tok, err)
	}
}

func TestAddAccountDoesNotOverwriteExistingInitLua(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	initPath := filepath.Join(dir, "init.lua")
	original := "return { theme = \"dracula\" }\n"
	if err := os.WriteFile(initPath, []byte(original), 0o600); err != nil {
		t.Fatalf("seed init.lua: %v", err)
	}

	out := captureStdout(t, func() {
		err := addAccount("gmail", "me@example.com", "client-abc", "", dir,
			fakeAuthenticate(&oauth2.Token{AccessToken: "at"}, nil), neverOpenBrowser)
		if err != nil {
			t.Fatalf("addAccount: %v", err)
		}
	})

	got, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != original {
		t.Fatalf("init.lua was modified, want it left untouched:\n%s", got)
	}
	if !strings.Contains(out, `id = "me@example.com"`) || !strings.Contains(out, `client_id = "client-abc"`) {
		t.Fatalf("stdout = %q, want the account entry printed for the user to paste in", out)
	}

	// The token should still be saved even though the config wasn't touched.
	if tok, _ := auth.LoadToken("me@example.com"); tok == nil {
		t.Fatal("token should still be saved when init.lua already exists")
	}
}

func TestAddAccountAuthenticateFailure(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()

	err := addAccount("gmail", "me@example.com", "client-abc", "", dir,
		fakeAuthenticate(nil, errors.New("user denied")), neverOpenBrowser)
	if err == nil {
		t.Fatal("addAccount: expected error when authenticate fails, got nil")
	}

	if _, statErr := os.Stat(filepath.Join(dir, "init.lua")); statErr == nil {
		t.Fatal("init.lua should not be written when authentication fails")
	}
}
