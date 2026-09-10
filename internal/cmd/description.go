package cmd

import "github.com/alecthomas/kong"

var Description = kong.Description(`Sift is a terminal email client for Gmail and Outlook.

	Running sift with no arguments launches the TUI. Everything else is a
	CLI-only subcommand.
	`)
