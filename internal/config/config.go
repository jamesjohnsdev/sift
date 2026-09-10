// Package config loads sift's Lua configuration: a single init.lua that
// returns a table of theme and keymap overrides, layered on top of
// sensible built-in defaults.
package config

import (
	"errors"
	"fmt"
	"os"
)

type Config struct {
	Theme    Theme
	Keymap   Keymap
	Accounts []Account
}

func Default() Config {
	return Config{
		Theme:  TokyoNight(),
		Keymap: DefaultKeymap(),
	}
}

// Load reads and runs the Lua config at path, merging it onto Default().
// A missing file is not an error - it just means no overrides. Any other
// failure (bad Lua, wrong shape) is returned alongside a fully usable
// Default() config, so callers can warn loudly and keep running rather
// than fail to start.
func Load(path string) (Config, error) {
	cfg := Default()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}

	tbl, err := runInit(path)
	if err != nil {
		return cfg, err
	}

	if bad := unknownField(tbl, topLevelFields); bad != "" {
		return cfg, fmt.Errorf("%s: unknown field %q", path, bad)
	}

	if v, ok := tableString(tbl, "theme"); ok {
		theme, ok := lookupTheme(v)
		if !ok {
			return cfg, fmt.Errorf("theme %q is not a built-in theme", v)
		}
		cfg.Theme = theme
	} else if t, ok := tableTable(tbl, "theme"); ok {
		custom, err := parseTheme(t)
		if err != nil {
			return cfg, err
		}
		cfg.Theme = cfg.Theme.overrideFrom(custom)
	}

	if t, ok := tableTable(tbl, "keymap"); ok {
		km, err := parseKeymap(t)
		if err != nil {
			return cfg, err
		}
		cfg.Keymap = cfg.Keymap.overrideFrom(km)
	}

	if t, ok := tableTable(tbl, "accounts"); ok {
		accounts, err := parseAccounts(t)
		if err != nil {
			return cfg, err
		}
		cfg.Accounts = accounts
	}

	return cfg, nil
}
