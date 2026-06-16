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
	helpBoxStyle   = lipgloss.NewStyle().Padding(1, 3)
	helpTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	helpKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
)

type HelpBinding struct {
	Keys        string
	Description string
}

type Help struct {
	config   HelpConfig
	visible  bool
	bindings []HelpBinding
}

func NewHelp(config HelpConfig) Help {
	return Help{config: config}
}

func (h Help) WithBindings(bindings []HelpBinding) Help {
	h.bindings = bindings

	return h
}

func (h Help) Visible() bool { return h.visible }

func (h Help) Toggle() Help {
	if h.config.Enabled {
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
		if width := lipgloss.Width(binding.Keys); width > keyWidth {
			keyWidth = width
		}
	}

	rows := make([]string, len(h.bindings))

	for i, binding := range h.bindings {
		padding := strings.Repeat(" ", keyWidth-lipgloss.Width(binding.Keys))

		rows[i] = helpKeyStyle.Render(binding.Keys) + padding + "   " + subtleStyle.Render(binding.Description)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		helpTitleStyle.Render("keybindings"),
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)

	return helpBoxStyle.Render(content)
}
