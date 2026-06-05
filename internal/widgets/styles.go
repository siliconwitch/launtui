package widgets

import "github.com/charmbracelet/lipgloss"

var (
	launcherColor = lipgloss.Color("4")
	clockColor    = lipgloss.Color("5")
	batteryColor  = lipgloss.Color("2")
	faintColor    = lipgloss.Color("8")

	subtleStyle = lipgloss.NewStyle().Foreground(faintColor)
)

func runeLen(s string) int {
	return len([]rune(s))
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}

	r := []rune(s)

	if len(r) <= w {
		return s
	}

	if w == 1 {
		return "…"
	}

	return string(r[:w-1]) + "…"
}
