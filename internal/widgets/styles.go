package widgets

import "github.com/charmbracelet/lipgloss"

// Shared widget palette — muted greys with a single bright accent, in the
// spirit of bluetui/impala.
var (
	accent   = lipgloss.Color("#89b4fa")
	fgBright = lipgloss.Color("#cdd6f4")
	fgFaint  = lipgloss.Color("#6c7086")

	clockTimeStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	clockDateStyle = lipgloss.NewStyle().Foreground(fgFaint)

	nameStyle    = lipgloss.NewStyle().Foreground(fgBright)
	selNameStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	selBarStyle  = lipgloss.NewStyle().Foreground(accent)
	subtleStyle  = lipgloss.NewStyle().Foreground(fgFaint)
)

// runeLen / truncate are ASCII-width approximations — fine for app names and
// short comments, and avoids a wcwidth dependency.
func runeLen(s string) int { return len([]rune(s)) }

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
