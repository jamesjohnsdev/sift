package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns sift's config directory (creating it if needed): the OS
// config dir joined with "sift", where init.lua and sift.db live.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	dir := filepath.Join(base, "sift")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}
