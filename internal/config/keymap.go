package config

// Keymap maps each action to the keys that trigger it. Values mirror what
// bubbles/key.WithKeys accepts (e.g. "j", "up", "ctrl+c").
type Keymap struct {
	Up         []string
	Down       []string
	Top        []string
	Bottom     []string
	FocusLeft  []string
	FocusRight []string
	Quit       []string
}

func DefaultKeymap() Keymap {
	return Keymap{
		Up:         []string{"k", "up"},
		Down:       []string{"j", "down"},
		Top:        []string{"g"},
		Bottom:     []string{"G"},
		FocusLeft:  []string{"h", "left"},
		FocusRight: []string{"l", "right"},
		Quit:       []string{"q", "ctrl+c"},
	}
}

// overrideFrom returns k with any non-empty field of o applied on top, so a
// Lua keymap table only needs to set the bindings it wants to change.
func (k Keymap) overrideFrom(o Keymap) Keymap {
	if len(o.Up) > 0 {
		k.Up = o.Up
	}
	if len(o.Down) > 0 {
		k.Down = o.Down
	}
	if len(o.Top) > 0 {
		k.Top = o.Top
	}
	if len(o.Bottom) > 0 {
		k.Bottom = o.Bottom
	}
	if len(o.FocusLeft) > 0 {
		k.FocusLeft = o.FocusLeft
	}
	if len(o.FocusRight) > 0 {
		k.FocusRight = o.FocusRight
	}
	if len(o.Quit) > 0 {
		k.Quit = o.Quit
	}
	return k
}
