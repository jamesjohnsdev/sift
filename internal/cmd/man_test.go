package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/jamesjohnsdev/sift/internal/cmd"
)

func TestRenderManPage(t *testing.T) {
	k, err := kong.New(&cmd.CLI{}, kong.Name("sift"), cmd.Description)
	if err != nil {
		t.Fatalf("kong.New() error = %v", err)
	}

	dir := t.TempDir()
	dest, err := cmd.RenderManPage(k.Model, dir)
	if err != nil {
		t.Fatalf("RenderManPage() error = %v", err)
	}
	if dest != filepath.Join(dir, "sift.1") {
		t.Fatalf("dest = %q, want %s", dest, filepath.Join(dir, "sift.1"))
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	body := string(data)
	if !strings.Contains(body, ".nr HY 0") {
		t.Fatalf("man page missing hyphenation-disable register:\n%s", body)
	}
	if !strings.Contains(body, "SIFT") {
		t.Fatalf("man page missing expected title:\n%s", body)
	}
}
