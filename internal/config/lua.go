package config

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// runInit executes the Lua file at path and returns the table it returns
// (`return { ... }`), the convention this whole config is built around.
func runInit(path string) (*lua.LTable, error) {
	L := lua.NewState()
	defer L.Close()

	if err := L.DoFile(path); err != nil {
		return nil, fmt.Errorf("run %s: %w", path, err)
	}

	if L.GetTop() == 0 {
		return nil, fmt.Errorf("%s must end with `return { ... }`", path)
	}
	tbl, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("%s must return a table, got %s", path, L.Get(-1).Type())
	}
	return tbl, nil
}

func tableString(tbl *lua.LTable, key string) (string, bool) {
	v := tbl.RawGetString(key)
	s, ok := v.(lua.LString)
	if !ok {
		return "", false
	}
	return string(s), true
}

func tableTable(tbl *lua.LTable, key string) (*lua.LTable, bool) {
	v := tbl.RawGetString(key)
	t, ok := v.(*lua.LTable)
	return t, ok
}

// unknownField returns the first key in tbl not present in allowed, or ""
// if every key is known. Catches typos (e.g. `imap_prot`) that would
// otherwise be silently ignored instead of erroring.
func unknownField(tbl *lua.LTable, allowed map[string]bool) string {
	bad := ""
	tbl.ForEach(func(k, _ lua.LValue) {
		if bad != "" {
			return
		}
		ks, ok := k.(lua.LString)
		if !ok {
			return
		}
		if !allowed[string(ks)] {
			bad = string(ks)
		}
	})
	return bad
}

var themeFields = map[string]bool{
	"bg": true, "fg": true, "muted": true, "border": true,
	"accent": true, "select_bg": true, "status_bg": true,
}

var keymapFields = map[string]bool{
	"up": true, "down": true, "top": true, "bottom": true,
	"focus_left": true, "focus_right": true, "quit": true,
	"compose": true, "reply": true,
}

var accountFields = map[string]bool{
	"id": true, "kind": true, "email": true, "client_id": true,
}

var topLevelFields = map[string]bool{
	"theme": true, "keymap": true, "accounts": true,
}

// tableStringSlice reads a Lua array of strings, e.g. {"j", "down"}.
func tableStringSlice(tbl *lua.LTable, key string) ([]string, error) {
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return nil, nil
	}
	t, ok := v.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("%s must be a list of strings, got %s", key, v.Type())
	}

	var out []string
	var rangeErr error
	t.ForEach(func(_, val lua.LValue) {
		s, ok := val.(lua.LString)
		if !ok {
			rangeErr = fmt.Errorf("%s entries must be strings, got %s", key, val.Type())
			return
		}
		out = append(out, string(s))
	})
	return out, rangeErr
}

func parseTheme(tbl *lua.LTable) (Theme, error) {
	if bad := unknownField(tbl, themeFields); bad != "" {
		return Theme{}, fmt.Errorf("theme: unknown field %q", bad)
	}

	var t Theme
	fields := map[string]*string{
		"bg": &t.Bg, "fg": &t.Fg, "muted": &t.Muted, "border": &t.Border,
		"accent": &t.Accent, "select_bg": &t.SelectBg, "status_bg": &t.StatusBg,
	}
	for key, dst := range fields {
		if v, ok := tableString(tbl, key); ok {
			*dst = v
		} else if tbl.RawGetString(key) != lua.LNil {
			return Theme{}, fmt.Errorf("theme.%s must be a string", key)
		}
	}
	return t, nil
}

// parseAccounts reads an array of account tables, e.g.
// { { id = "work", kind = "gmail", email = "me@x.com", client_id = "..." } }.
// Errors are reported with a field path (e.g. "accounts[2].kind") so a typo
// or wrong-typed field points straight at the offending entry.
func parseAccounts(tbl *lua.LTable) ([]Account, error) {
	var accounts []Account
	var rangeErr error
	tbl.ForEach(func(k, v lua.LValue) {
		if rangeErr != nil {
			return
		}
		path := fmt.Sprintf("accounts[%s]", k.String())

		t, ok := v.(*lua.LTable)
		if !ok {
			rangeErr = fmt.Errorf("%s: must be a table, got %s", path, v.Type())
			return
		}
		if bad := unknownField(t, accountFields); bad != "" {
			rangeErr = fmt.Errorf("%s: unknown field %q", path, bad)
			return
		}

		a := Account{}
		id, ok := tableString(t, "id")
		if !ok {
			rangeErr = fmt.Errorf("%s: missing required field id", path)
			return
		}
		a.ID = id

		fields := map[string]*string{"kind": &a.Kind, "email": &a.Email, "client_id": &a.ClientID}
		for key, dst := range fields {
			if v, ok := tableString(t, key); ok {
				*dst = v
			} else if t.RawGetString(key) != lua.LNil {
				rangeErr = fmt.Errorf("%s.%s: must be a string, got %s", path, key, t.RawGetString(key).Type())
				return
			}
		}
		if a.Kind != "" && a.Kind != "gmail" && a.Kind != "outlook" {
			rangeErr = fmt.Errorf("%s.kind: must be \"gmail\" or \"outlook\", got %q", path, a.Kind)
			return
		}
		accounts = append(accounts, a)
	})
	return accounts, rangeErr
}

func parseKeymap(tbl *lua.LTable) (Keymap, error) {
	if bad := unknownField(tbl, keymapFields); bad != "" {
		return Keymap{}, fmt.Errorf("keymap: unknown field %q", bad)
	}

	var k Keymap
	fields := map[string]*[]string{
		"up": &k.Up, "down": &k.Down, "top": &k.Top, "bottom": &k.Bottom,
		"focus_left": &k.FocusLeft, "focus_right": &k.FocusRight, "quit": &k.Quit,
		"compose": &k.Compose, "reply": &k.Reply,
	}
	for key, dst := range fields {
		keys, err := tableStringSlice(tbl, key)
		if err != nil {
			return Keymap{}, fmt.Errorf("keymap.%w", err)
		}
		*dst = keys
	}
	return k, nil
}
