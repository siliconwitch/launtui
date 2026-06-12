package widgets

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

type Mode interface {
	Name() string
	Hotkey() string
	Enabled() bool
	Init() tea.Cmd
	Update(tea.Msg) (Mode, tea.Cmd)
	SetQuery(query string) Mode
	HasResults() bool
	MoveUp() Mode
	MoveDown() Mode
	Activate() tea.Cmd
	View(width, rows int) string
}

type RequestQuitMsg struct{}

type AppClosingMsg struct{}

type StrongMatcher interface {
	StrongMatch() bool
}

type HistoryEditor interface {
	DeleteSelectedHistory() (Mode, tea.Cmd, bool)
	ClearHistory() (Mode, tea.Cmd)
}

var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

type list[T any] struct {
	key      func(T) string
	items    []T
	filtered []T
	query    string
	cursor   int
	loaded   bool
}

func newList[T any](key func(T) string) list[T] {
	return list[T]{key: key}
}

func (l *list[T]) setItems(items []T) {
	l.items = items
	l.loaded = true
	l.refilter()
}

func (l *list[T]) setQuery(query string) {
	l.query = query
	l.cursor = 0
	l.refilter()
}

func (l *list[T]) refilter() {
	query := strings.TrimSpace(l.query)

	if query == "" {
		l.filtered = l.items
	} else {
		names := make([]string, len(l.items))

		for i, item := range l.items {
			names[i] = l.key(item)
		}

		matches := fuzzy.Find(query, names)
		l.filtered = make([]T, len(matches))

		for i, match := range matches {
			l.filtered[i] = l.items[match.Index]
		}
	}

	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

func (l *list[T]) moveUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

func (l *list[T]) moveDown() {
	if l.cursor < len(l.filtered)-1 {
		l.cursor++
	}
}

func (l list[T]) hasResults() bool {
	return l.loaded && len(l.filtered) > 0
}

func (l list[T]) selected() (T, bool) {
	if len(l.filtered) == 0 {
		var zero T

		return zero, false
	}

	return l.filtered[l.cursor], true
}

func (l list[T]) view(width, rows int, render func(item T, selected bool, width int) string) string {
	start, end := visibleRange(l.cursor, rows, len(l.filtered))

	lines := make([]string, 0, end-start)

	for i := start; i < end; i++ {
		lines = append(lines, render(l.filtered[i], i == l.cursor, width))
	}

	return strings.Join(lines, "\n")
}

func renderRow(accent lipgloss.Color, selected bool, name, sub string) string {
	if selected {
		accentStyle := lipgloss.NewStyle().Foreground(accent)

		return accentStyle.Render("▌ ") + accentStyle.Bold(true).Render(name) + sub
	}

	return "  " + name + sub
}

func renderHistoryRow(accent lipgloss.Color, selected bool, name string) string {
	if selected {
		return renderRow(accent, true, name, "")
	}

	return "  " + subtleStyle.Render(name)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= w {
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

func removeAt[T any](entries []T, index int) []T {
	return append(append([]T{}, entries[:index]...), entries[index+1:]...)
}

func prependCapped[T any](entries []T, entry T, limit int, duplicate func(T) bool) []T {
	kept := make([]T, 0, len(entries)+1)
	kept = append(kept, entry)

	for _, existing := range entries {
		if duplicate == nil || !duplicate(existing) {
			kept = append(kept, existing)
		}
	}

	if limit > 0 && len(kept) > limit {
		kept = kept[:limit]
	}

	return kept
}
