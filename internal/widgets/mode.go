package widgets

import tea "github.com/charmbracelet/bubbletea"

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
