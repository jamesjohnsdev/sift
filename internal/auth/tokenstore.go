package auth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const keyringService = "sift"

// SaveToken stores tok in the OS keychain (Keychain on macOS, Secret
// Service on Linux, Credential Manager on Windows) under key - never on
// disk or in the Lua config. key is caller-defined (e.g. an account ID);
// this package has no notion of accounts itself.
func SaveToken(key string, tok *oauth2.Token) error {
	data, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := keyring.Set(keyringService, key, string(data)); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

// LoadToken returns nil, nil if no token has been saved under key yet.
func LoadToken(key string) (*oauth2.Token, error) {
	data, err := keyring.Get(keyringService, key)
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

func DeleteToken(key string) error {
	if err := keyring.Delete(keyringService, key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete token: %w", err)
	}
	return nil
}
