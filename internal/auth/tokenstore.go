package auth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

const keyringService = "sift"

// SaveToken stores account's token in the OS keychain (Keychain on macOS,
// Secret Service on Linux, Credential Manager on Windows) - never on disk
// or in the Lua config.
func SaveToken(account provider.AccountID, tok *oauth2.Token) error {
	data, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := keyring.Set(keyringService, string(account), string(data)); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

// LoadToken returns nil, nil if no token has been saved for account yet.
func LoadToken(account provider.AccountID) (*oauth2.Token, error) {
	data, err := keyring.Get(keyringService, string(account))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load token: %w", err)
	}

	var tok oauth2.Token
	if err := json.Unmarshal([]byte(data), &tok); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return &tok, nil
}

func DeleteToken(account provider.AccountID) error {
	if err := keyring.Delete(keyringService, string(account)); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete token: %w", err)
	}
	return nil
}
