package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/alecthomas/kong"

	"github.com/jamesjohnsdev/sift/internal/cmd"
	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/storage"
	"github.com/jamesjohnsdev/sift/internal/syncengine"
	"github.com/jamesjohnsdev/sift/internal/tui"
)

// version is set via -ldflags at build time; "dev" for local builds.
var version = "dev"

func main() {
	if len(os.Args) == 1 {
		if err := runTUI(); err != nil {
			fmt.Fprintln(os.Stderr, "sift:", err)
			os.Exit(1)
		}
		return
	}

	parser, err := kong.New(&cmd.CLI{}, kong.Name("sift"), cmd.Description, kong.Vars{
		"version": "sift " + version,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sift:", err)
		os.Exit(1)
	}

	ctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	if runtime.GOOS != "windows" && ctx.Command() != "man-install" {
		cmd.EnsureManPage(ctx.Model)
	}

	ctx.FatalIfErrorf(ctx.Run())
}

func runTUI() error {
	appDir, err := config.Dir()
	if err != nil {
		return err
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

	return tui.Run(cfg, store, mgr.Updates(), mgr.Send)
}
