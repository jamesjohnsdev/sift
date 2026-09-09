package main

import (
	"fmt"
	"os"

	"github.com/jamesjohnsdev/sift/internal/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "sift:", err)
		os.Exit(1)
	}
}
