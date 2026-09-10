package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"

	"github.com/jamesjohnsdev/sift/internal/auth"
	"github.com/jamesjohnsdev/sift/internal/config"
)

type AccountCmd struct {
	Add AccountAddCmd `cmd:"" help:"Authenticate a new account and save its token"`
}

type AccountAddCmd struct {
	Kind     string `arg:"" enum:"gmail,outlook" help:"Provider kind (gmail or outlook)"`
	Email    string `required:"" help:"Account email address"`
	ClientID string `required:"" name:"client-id" help:"OAuth client ID for your own app registration"`
	ID       string `help:"Account id for the keychain entry and config; defaults to the email address"`
}

func (c *AccountAddCmd) Run() error {
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	return addAccount(c.Kind, c.Email, c.ClientID, c.ID, dir, auth.Authenticate, auth.OpenBrowser)
}

// authenticateFunc matches auth.Authenticate's signature; a parameter here
// so tests can substitute a fake instead of running a real OAuth flow.
type authenticateFunc func(ctx context.Context, cfg auth.ProviderConfig, openBrowser func(string) error) (*oauth2.Token, error)

func addAccount(kind, email, clientID, id, dir string, authenticate authenticateFunc, openBrowser func(string) error) error {
	if id == "" {
		id = email
	}

	var oauthCfg auth.ProviderConfig
	switch kind {
	case "gmail":
		oauthCfg = auth.Gmail(clientID)
	case "outlook":
		oauthCfg = auth.Outlook(clientID)
	default:
		return fmt.Errorf("unknown kind %q", kind)
	}

	fmt.Println("opening browser to authenticate, complete login there...")
	tok, err := authenticate(context.Background(), oauthCfg, openBrowser)
	if err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	if err := auth.SaveToken(id, tok); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	fmt.Println("token saved to OS keychain")

	entry := fmt.Sprintf("{ id = %q, kind = %q, email = %q, client_id = %q }", id, kind, email, clientID)
	initPath := filepath.Join(dir, "init.lua")

	// Only auto-write when there's no config yet: rewriting an existing
	// init.lua risks clobbering the user's theme/keymap/other accounts
	// without a real Lua parser+printer round trip, which we don't have.
	if _, err := os.Stat(initPath); errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf("sift.setup({\n\taccounts = {\n\t\t%s,\n\t},\n})\n", entry)
		if err := os.WriteFile(initPath, []byte(content), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", initPath, err)
		}
		if _, err := config.Load(initPath); err != nil {
			return fmt.Errorf("wrote %s but it failed to load back: %w", initPath, err)
		}
		fmt.Printf("wrote account to %s\n", initPath)
		return nil
	}

	fmt.Println()
	fmt.Println("account authenticated. add this to the `accounts` list in", initPath+":")
	fmt.Println()
	fmt.Println("  " + entry + ",")
	return nil
}
