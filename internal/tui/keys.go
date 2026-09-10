package tui

import (
	"charm.land/bubbles/v2/key"

	"github.com/jamesjohnsdev/sift/internal/config"
)

// KeyMap is a plain struct of bindings built from config.Keymap.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Top        key.Binding // gg
	Bottom     key.Binding // G
	FocusLeft  key.Binding
	FocusRight key.Binding
	Quit       key.Binding
	Compose    key.Binding
	Reply      key.Binding
}

func newKeyMap(km config.Keymap) KeyMap {
	bind := func(keys []string, help string) key.Binding {
		return key.NewBinding(key.WithKeys(keys...), key.WithHelp(keys[0], help))
	}
	return KeyMap{
		Up:         bind(km.Up, "up"),
		Down:       bind(km.Down, "down"),
		Top:        bind(km.Top, "top"),
		Bottom:     bind(km.Bottom, "bottom"),
		FocusLeft:  bind(km.FocusLeft, "focus left"),
		FocusRight: bind(km.FocusRight, "focus right"),
		Quit:       bind(km.Quit, "quit"),
		Compose:    bind(km.Compose, "compose"),
		Reply:      bind(km.Reply, "reply"),
	}
}
