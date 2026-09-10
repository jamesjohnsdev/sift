package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	path := writeInit(t, `sift.setup({ theme = "dracula" })`)

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
	path := writeInit(t, `sift.setup({ theme = { accent = "#ff0000" } })`)

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
	path := writeInit(t, `sift.setup({ theme = "not-a-theme" })`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error for unknown theme name, got nil")
	}
}

func TestLoadKeymapOverridesOnlyGivenActions(t *testing.T) {
	path := writeInit(t, `sift.setup({ keymap = { quit = {"x"} } })`)

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

func TestLoadKeymapComposeAndReply(t *testing.T) {
	path := writeInit(t, `sift.setup({ keymap = { compose = {"n"}, reply = {"shift+r"} } })`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Keymap.Compose) != 1 || cfg.Keymap.Compose[0] != "n" {
		t.Fatalf("Keymap.Compose = %v, want [n]", cfg.Keymap.Compose)
	}
	if len(cfg.Keymap.Reply) != 1 || cfg.Keymap.Reply[0] != "shift+r" {
		t.Fatalf("Keymap.Reply = %v, want [shift+r]", cfg.Keymap.Reply)
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

func TestLoadMustCallSetup(t *testing.T) {
	path := writeInit(t, `local x = 1`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error when script never calls sift.setup, got nil")
	}
}

func TestLoadSetupArgMustBeTable(t *testing.T) {
	path := writeInit(t, `sift.setup("not a table")`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error when sift.setup is called with a non-table, got nil")
	}
}

func TestLoadAccounts(t *testing.T) {
	path := writeInit(t, `sift.setup({
		accounts = {
			{ id = "work", kind = "gmail", email = "me@example.com", client_id = "abc" },
			{ id = "personal", kind = "outlook", email = "me@outlook.com" },
		},
	})`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Accounts) != 2 {
		t.Fatalf("Accounts = %+v, want 2", cfg.Accounts)
	}
	want := Account{ID: "work", Kind: "gmail", Email: "me@example.com", ClientID: "abc"}
	if cfg.Accounts[0] != want {
		t.Fatalf("Accounts[0] = %+v, want %+v", cfg.Accounts[0], want)
	}
	if cfg.Accounts[1].ID != "personal" || cfg.Accounts[1].ClientID != "" {
		t.Fatalf("Accounts[1] = %+v", cfg.Accounts[1])
	}
}

func TestLoadAccountMissingID(t *testing.T) {
	path := writeInit(t, `sift.setup({ accounts = { { kind = "gmail" } } })`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error for account missing id, got nil")
	}
}

func TestLoadAccountWrongFieldType(t *testing.T) {
	// A table where a string is expected (e.g. a placeholder like
	// { "..." } left in client_id) must fail loudly, not silently
	// coerce to an empty string.
	path := writeInit(t, `sift.setup({ accounts = { { id = "x", client_id = { "..." } } } })`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error for non-string client_id, got nil")
	}
}

func TestLoadAccountUnknownField(t *testing.T) {
	// A typo'd field name (e.g. "knd" instead of "kind") must error, not
	// silently be ignored.
	path := writeInit(t, `sift.setup({ accounts = { { id = "x", knd = "gmail" } } })`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for unknown account field, got nil")
	}
	if !strings.Contains(err.Error(), "accounts[1]") || !strings.Contains(err.Error(), "knd") {
		t.Fatalf("Load: error = %q, want it to name accounts[1] and the field \"knd\"", err.Error())
	}
}

func TestLoadAccountUnknownKind(t *testing.T) {
	path := writeInit(t, `sift.setup({ accounts = { { id = "x", kind = "yahoo" } } })`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected error for unknown account kind, got nil")
	}
}

func TestLoadTopLevelUnknownField(t *testing.T) {
	path := writeInit(t, `sift.setup({ theeme = "dracula" })`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for unknown top-level field, got nil")
	}
	if !strings.Contains(err.Error(), "theeme") {
		t.Fatalf("Load: error = %q, want it to name the field \"theeme\"", err.Error())
	}
}

func TestLoadThemeUnknownField(t *testing.T) {
	path := writeInit(t, `sift.setup({ theme = { accnt = "#ff0000" } })`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for unknown theme field, got nil")
	}
	if !strings.Contains(err.Error(), "accnt") {
		t.Fatalf("Load: error = %q, want it to name the field \"accnt\"", err.Error())
	}
}

func TestLoadKeymapUnknownField(t *testing.T) {
	path := writeInit(t, `sift.setup({ keymap = { qwit = {"x"} } })`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for unknown keymap field, got nil")
	}
	if !strings.Contains(err.Error(), "qwit") {
		t.Fatalf("Load: error = %q, want it to name the field \"qwit\"", err.Error())
	}
}
