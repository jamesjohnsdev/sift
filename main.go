package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/sift/internal/config"
	"github.com/jamesjohnsdev/sift/internal/tui"
)

func main() {
	cfg := config.Default()
	if dir, err := os.UserConfigDir(); err == nil {
		loaded, err := config.Load(filepath.Join(dir, "sift", "init.lua"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "sift: config error, using defaults:", err)
		}
		cfg = loaded
	}

	if err := tui.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "sift:", err)
		os.Exit(1)
	}
}
