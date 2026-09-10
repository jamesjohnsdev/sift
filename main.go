package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/storage"
	"github.com/jamesjohnsdev/sift/internal/syncengine"
	"github.com/jamesjohnsdev/sift/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sift:", err)
		os.Exit(1)
	}
}

func run() error {
	appDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}
	appDir = filepath.Join(appDir, "sift")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfg, err := config.Load(filepath.Join(appDir, "init.lua"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "sift: config error, using defaults:", err)
	}

	store, err := storage.Open(filepath.Join(appDir, "sift.db"))
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "sift: close storage:", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := syncengine.New(store, cfg.Accounts, nil)
	mgr.Start(ctx)

	return tui.Run(cfg, store)
}
