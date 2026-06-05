package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HelpConfig struct {
	Enabled bool `toml:"enabled"`
}

func (HelpConfig) SectionName() string { return "help" }

func DefaultHelpConfig() HelpConfig {
	return HelpConfig{Enabled: true}
}

var (
	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(launcherColor).
			Padding(1, 3)
	helpTitleStyle = lipgloss.NewStyle().Foreground(launcherColor).Bold(true)
	helpKeyStyle   = lipgloss.NewStyle().Foreground(clockColor).Bold(true)
	helpDescStyle  = lipgloss.NewStyle().Foreground(faintColor)
)

type helpBinding struct {
	keys string
	desc string
}

var helpBindings = []helpBinding{
	{"type", "filter the list"},
	{"↑ / ↓", "move selection"},
	{"enter", "launch selection"},
	{"ctrl+h", "toggle this help"},
	{"esc", "quit"},
}

type Help struct {
	cfg     HelpConfig
	visible bool
}

func NewHelp(cfg HelpConfig) Help {
	return Help{cfg: cfg}
}

func (h Help) Enabled() bool { return h.cfg.Enabled }

func (h Help) Visible() bool { return h.visible }

func (h Help) Toggle() Help {
	if h.cfg.Enabled {
		h.visible = !h.visible
	}

	return h
}

func (h Help) Hide() Help {
	h.visible = false

	return h
}

func (h Help) View() string {
	keyWidth := 0

	for _, binding := range helpBindings {
		if width := runeLen(binding.keys); width > keyWidth {
			keyWidth = width
		}
	}

	rows := make([]string, len(helpBindings))

	for i, binding := range helpBindings {
		padding := strings.Repeat(" ", keyWidth-runeLen(binding.keys))

		rows[i] = helpKeyStyle.Render(binding.keys) + padding + "   " + helpDescStyle.Render(binding.desc)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		helpTitleStyle.Render("keybindings"),
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)

	return helpBoxStyle.Render(content)
}
