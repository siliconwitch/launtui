package widgets

import tea "github.com/charmbracelet/bubbletea"

type ClipboardConfig struct {
	Enabled bool `toml:"enabled"`
}

func (ClipboardConfig) SectionName() string { return "clipboard" }

func DefaultClipboardConfig() ClipboardConfig {
	return ClipboardConfig{Enabled: true}
}

type Clipboard struct {
	cfg ClipboardConfig
}

func NewClipboard(cfg ClipboardConfig) Clipboard {
	return Clipboard{cfg: cfg}
}

func (Clipboard) Name() string                     { return "Clip" }
func (Clipboard) Hotkey() string                   { return "ctrl+v" }
func (c Clipboard) Enabled() bool                  { return c.cfg.Enabled }
func (Clipboard) Init() tea.Cmd                    { return nil }
func (c Clipboard) Update(tea.Msg) (Mode, tea.Cmd) { return c, nil }
func (c Clipboard) SetQuery(string) Mode           { return c }
func (Clipboard) HasResults() bool                 { return false }
func (c Clipboard) MoveUp() Mode                   { return c }
func (c Clipboard) MoveDown() Mode                 { return c }
func (Clipboard) Activate() tea.Cmd                { return nil }

func (Clipboard) View(width, rows int) string {
	return subtleStyle.Render("clipboard history coming soon")
}
