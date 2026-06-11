package widgets

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	launcherColor  = lipgloss.Color("4")
	clockColor     = lipgloss.Color("5")
	batteryColor   = lipgloss.Color("2")
	faintColor     = lipgloss.Color("8")
	passwordColor  = lipgloss.Color("3")
	clipboardColor = lipgloss.Color("6")
	projectColor   = lipgloss.Color("2")
	webColor       = lipgloss.Color("12")
	cleanColor     = lipgloss.Color("2")
	dirtyColor     = lipgloss.Color("3")
	errorColor     = lipgloss.Color("1")

	nameStyle   = lipgloss.NewStyle()
	subtleStyle = lipgloss.NewStyle().Foreground(faintColor)
	errorStyle  = lipgloss.NewStyle().Foreground(errorColor)
)

func displayWidth(s string) int {
	return lipgloss.Width(s)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}

	if displayWidth(s) <= w {
		return s
	}

	if w == 1 {
		return "…"
	}

	return ansi.Truncate(s, w-1, "") + "…"
}

func visibleRange(cursor, rows, count int) (int, int) {
	if rows < 1 {
		rows = 1
	}

	start := 0

	if cursor >= rows {
		start = cursor - rows + 1
	}

	return start, min(start+rows, count)
}

func timeAgo(unix, now int64) string {
	elapsed := now - unix

	switch {
	case elapsed < 60:
		return "now"
	case elapsed < 3600:
		return strconv.FormatInt(elapsed/60, 10) + "m"
	case elapsed < 86400:
		return strconv.FormatInt(elapsed/3600, 10) + "h"
	default:
		return strconv.FormatInt(elapsed/86400, 10) + "d"
	}
}
