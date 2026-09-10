// Package cmd is sift's CLI surface (Kong-driven), separate from the TUI
// itself (internal/tui). Running sift with no arguments launches the TUI
// directly, bypassing this package entirely - see main.go.
package cmd

import "github.com/alecthomas/kong"

type CLI struct {
	Version kong.VersionFlag `help:"Print version and exit"`

	ManInstall ManInstallCmd `cmd:"" name:"man-install" help:"Install man page for local use"`
}
