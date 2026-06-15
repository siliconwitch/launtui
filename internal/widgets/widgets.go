package widgets

import (
	"strconv"
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
	Rows() []Row
	Accent() lipgloss.Color
	Activate(index int) tea.Cmd
	Status() string
}

type RequestQuitMsg struct{}

type AppClosingMsg struct{}

type StrongMatcher interface {
	StrongMatch() bool
}

type RowDeleter interface {
	DeleteRow(index int) (Mode, tea.Cmd)
	ClearRows() (Mode, tea.Cmd)
}

type Selectable interface {
	Select(index int) (Mode, tea.Cmd)
}

var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

type rowStyle int

const (
	rowPlain rowStyle = iota
	rowDim
	rowEmphasized
)

type Row struct {
	left      string
	right     string
	style     rowStyle
	Deletable bool
}

func HasResults(rows []Row) bool {
	for _, row := range rows {
		if row.style != rowDim {
			return true
		}
	}

	return false
}

func RenderResults(status string, rows []Row, accent lipgloss.Color, cursor, width, height int) string {
	var lines []string

	if status != "" {
		lines = append(lines, status)

		height -= lipgloss.Height(status)
	}

	if len(rows) > 0 && height > 0 {
		start, end := visibleRange(cursor, height, len(rows))

		for i := start; i < end; i++ {
			lines = append(lines, renderRow(accent, i == cursor, rows[i], width))
		}
	}

	return strings.Join(lines, "\n")
}

func renderRow(accent lipgloss.Color, selected bool, row Row, width int) string {
	avail := max(width-2, 1)
	name := row.left
	sub := ""

	if lipgloss.Width(name) > avail {
		name = truncate(name, avail)
	} else if row.right != "" {
		gap := avail - lipgloss.Width(name)

		if gap > 2 {
			right := truncate(row.right, gap-1)
			sub = strings.Repeat(" ", gap-lipgloss.Width(right)) + right
		}
	}

	if selected {
		accentStyle := lipgloss.NewStyle().Foreground(accent)

		return accentStyle.Render("▌ ") + accentStyle.Bold(true).Render(name) + sub
	}

	switch row.style {
	case rowDim:
		return "  " + subtleStyle.Render(name) + sub
	case rowEmphasized:
		return "  " + lipgloss.NewStyle().Foreground(accent).Bold(true).Render(name) + sub
	default:
		return "  " + name + sub
	}
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

func relativeAge(elapsed int64) string {
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

type list[T any] struct {
	key      func(T) string
	items    []T
	filtered []T
	query    string
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
	l.refilter()
}

func (l *list[T]) refilter() {
	query := strings.TrimSpace(l.query)

	if query == "" {
		l.filtered = l.items

		return
	}

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

func (l list[T]) rows(cell func(item T) Row) []Row {
	rows := make([]Row, len(l.filtered))

	for i, item := range l.filtered {
		rows[i] = cell(item)
	}

	return rows
}

func (l list[T]) at(index int) (T, bool) {
	if index < 0 || index >= len(l.filtered) {
		var zero T

		return zero, false
	}

	return l.filtered[index], true
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
