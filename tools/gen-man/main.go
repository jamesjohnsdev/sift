// Command gen-man renders sift's man page for a release build, which has
// no user home directory to install into (see cmd.RenderManPage).
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/jamesjohnsdev/sift/internal/cmd"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: gen-man <output-dir>")
		os.Exit(1)
	}

	parser, err := kong.New(&cmd.CLI{}, kong.Name("sift"), cmd.Description, kong.Vars{
		"version": "sift",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen-man:", err)
		os.Exit(1)
	}

	dest, err := cmd.RenderManPage(parser.Model, os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen-man:", err)
		os.Exit(1)
	}

	fmt.Println("wrote", dest)
}
