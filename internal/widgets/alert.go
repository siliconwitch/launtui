package widgets

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const alertWidth = 56

var (
	alertBoxStyle   = lipgloss.NewStyle().Padding(1, 3)
	alertTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
)

type Alert struct {
	message string
	visible bool
}

func NewAlert(message string) Alert {
	return Alert{message: message, visible: message != ""}
}

func (a Alert) Visible() bool { return a.visible }

func (a Alert) Hide() Alert {
	a.visible = false

	return a
}

func (a Alert) View() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		alertTitleStyle.Render("error"),
		"",
		ansi.Wrap(a.message, alertWidth, ""),
	)

	return alertBoxStyle.Render(content)
}
