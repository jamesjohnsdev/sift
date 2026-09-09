package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeInit(t *testing.T, lua string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "init.lua")
	if err := os.WriteFile(path, []byte(lua), 0o644); err != nil {
		t.Fatalf("write init.lua: %v", err)
	}
	return path
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.lua"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("Load(missing) = %+v, want Default()", cfg)
	}
}

func TestLoadBuiltinTheme(t *testing.T) {
	path := writeInit(t, `return { theme = "dracula" }`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != Dracula() {
		t.Fatalf("Theme = %+v, want Dracula()", cfg.Theme)
	}
	if !reflect.DeepEqual(cfg.Keymap, DefaultKeymap()) {
		t.Fatalf("Keymap should be untouched by a theme-only config")
	}
}

func TestLoadCustomThemeOverridesOnlyGivenFields(t *testing.T) {
	path := writeInit(t, `return { theme = { accent = "#ff0000" } }`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := TokyoNight()
	want.Accent = "#ff0000"
	if cfg.Theme != want {
		t.Fatalf("Theme = %+v, want %+v", cfg.Theme, want)
	}
}

func TestLoadUnknownThemeName(t *testing.T) {
	path := writeInit(t, `return { theme = "not-a-theme" }`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error for unknown theme name, got nil")
	}
}

func TestLoadKeymapOverridesOnlyGivenActions(t *testing.T) {
	path := writeInit(t, `return { keymap = { quit = {"x"} } }`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Keymap.Quit) != 1 || cfg.Keymap.Quit[0] != "x" {
		t.Fatalf("Keymap.Quit = %v, want [x]", cfg.Keymap.Quit)
	}
	if got := cfg.Keymap.Up; len(got) != 2 || got[0] != "k" {
		t.Fatalf("Keymap.Up should be untouched, got %v", got)
	}
}

func TestLoadInvalidLuaReturnsDefaultAndError(t *testing.T) {
	path := writeInit(t, `this is not valid lua {{{`)

	cfg, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for invalid Lua, got nil")
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("Load(invalid) = %+v, want Default() alongside the error", cfg)
	}
}

func TestLoadMustReturnATable(t *testing.T) {
	path := writeInit(t, `return "not a table"`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error when script doesn't return a table, got nil")
	}
}
