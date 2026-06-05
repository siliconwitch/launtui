package widgets

import tea "github.com/charmbracelet/bubbletea"

type PasswordsConfig struct {
	Enabled bool `toml:"enabled"`
}

func (PasswordsConfig) SectionName() string { return "passwords" }

func DefaultPasswordsConfig() PasswordsConfig {
	return PasswordsConfig{Enabled: true}
}

type Passwords struct {
	cfg PasswordsConfig
}

func NewPasswords(cfg PasswordsConfig) Passwords {
	return Passwords{cfg: cfg}
}

func (Passwords) Name() string                     { return "Pass" }
func (Passwords) Hotkey() string                   { return "ctrl+p" }
func (p Passwords) Enabled() bool                  { return p.cfg.Enabled }
func (Passwords) Init() tea.Cmd                    { return nil }
func (p Passwords) Update(tea.Msg) (Mode, tea.Cmd) { return p, nil }
func (p Passwords) SetQuery(string) Mode           { return p }
func (Passwords) HasResults() bool                 { return false }
func (p Passwords) MoveUp() Mode                   { return p }
func (p Passwords) MoveDown() Mode                 { return p }
func (Passwords) Activate() tea.Cmd                { return nil }

func (Passwords) View(width, rows int) string {
	return subtleStyle.Render("password manager coming soon")
}
