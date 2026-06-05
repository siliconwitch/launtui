package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/siliconwitch/launtui/internal/widgets"
)

var (
	appStyle     = lipgloss.NewStyle().Padding(0, 1)
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")).Bold(true)
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a"))
)

type App struct {
	clock    widgets.Clock
	launcher widgets.Launcher

	width  int
	height int
}

func New() (App, error) {
	launcherCfg := widgets.DefaultLauncherConfig()
	clockCfg := widgets.DefaultClockConfig()

	err := Load(&launcherCfg, &clockCfg)

	return App{
		clock:    widgets.NewClock(clockCfg),
		launcher: widgets.NewLauncher(launcherCfg),
	}, err
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.clock.Init(), a.launcher.Init())
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return a, tea.Quit
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	a.clock, cmd = a.clock.Update(msg)
	cmds = append(cmds, cmd)

	a.launcher, cmd = a.launcher.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}

	tuiWidth := max(1, a.width)
	tuiHeight := max(1, a.height)
	contentWidth := max(1, a.width-2)

	left := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("launtui"),
		a.launcher.InputView(contentWidth/2),
	)

	header := spread(contentWidth, left, a.clock.View())

	divider := dividerStyle.Render(strings.Repeat("─", contentWidth))

	rows := tuiHeight - lipgloss.Height(header) - 1

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		divider,
		a.launcher.ListView(contentWidth, rows),
	)

	return appStyle.Width(tuiWidth).Height(tuiHeight).Render(body)
}

func spread(width int, left, right string) string {
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right)
}
