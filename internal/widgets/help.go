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

type HelpBinding struct {
	Keys string
	Desc string
}

type Help struct {
	cfg      HelpConfig
	visible  bool
	bindings []HelpBinding
}

func NewHelp(cfg HelpConfig) Help {
	return Help{cfg: cfg}
}

func (h Help) WithBindings(bindings []HelpBinding) Help {
	h.bindings = bindings

	return h
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

	for _, binding := range h.bindings {
		if width := displayWidth(binding.Keys); width > keyWidth {
			keyWidth = width
		}
	}

	rows := make([]string, len(h.bindings))

	for i, binding := range h.bindings {
		padding := strings.Repeat(" ", keyWidth-displayWidth(binding.Keys))

		rows[i] = helpKeyStyle.Render(binding.Keys) + padding + "   " + helpDescStyle.Render(binding.Desc)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		helpTitleStyle.Render("keybindings"),
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)

	return helpBoxStyle.Render(content)
}
