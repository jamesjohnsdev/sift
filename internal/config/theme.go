package config

// Theme is a Lua table of hex colors consumed by lipgloss. Fields are
// plain strings, not lipgloss.Color, so this package has no UI dependency.
type Theme struct {
	Bg       string
	Fg       string
	Muted    string
	Border   string
	Accent   string
	SelectBg string
	StatusBg string
}

func TokyoNight() Theme {
	return Theme{
		Bg:       "#1a1b26",
		Fg:       "#c0caf5",
		Muted:    "#565f89",
		Border:   "#414868",
		Accent:   "#7aa2f7",
		SelectBg: "#283457",
		StatusBg: "#24283b",
	}
}

func Catppuccin() Theme {
	return Theme{
		Bg:       "#1e1e2e",
		Fg:       "#cdd6f4",
		Muted:    "#6c7086",
		Border:   "#45475a",
		Accent:   "#89b4fa",
		SelectBg: "#313244",
		StatusBg: "#181825",
	}
}

func Dracula() Theme {
	return Theme{
		Bg:       "#282a36",
		Fg:       "#f8f8f2",
		Muted:    "#6272a4",
		Border:   "#44475a",
		Accent:   "#bd93f9",
		SelectBg: "#44475a",
		StatusBg: "#21222c",
	}
}

// builtinThemes are looked up by name for `theme = "catppuccin"` in Lua.
var builtinThemes = map[string]func() Theme{
	"tokyo-night": TokyoNight,
	"catppuccin":  Catppuccin,
	"dracula":     Dracula,
}

func lookupTheme(name string) (Theme, bool) {
	f, ok := builtinThemes[name]
	if !ok {
		return Theme{}, false
	}
	return f(), true
}

// overrideFrom returns t with any non-empty field of o applied on top, so a
// custom Lua theme table only needs to set the colors it wants to change.
func (t Theme) overrideFrom(o Theme) Theme {
	if o.Bg != "" {
		t.Bg = o.Bg
	}
	if o.Fg != "" {
		t.Fg = o.Fg
	}
	if o.Muted != "" {
		t.Muted = o.Muted
	}
	if o.Border != "" {
		t.Border = o.Border
	}
	if o.Accent != "" {
		t.Accent = o.Accent
	}
	if o.SelectBg != "" {
		t.SelectBg = o.SelectBg
	}
	if o.StatusBg != "" {
		t.StatusBg = o.StatusBg
	}
	return t
}
