package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/siliconwitch/launtui/internal/widgets"
)

var (
	appStyle     = lipgloss.NewStyle().Padding(0, 1)
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type App struct {
	clock    widgets.Clock
	battery  widgets.Battery
	launcher widgets.Launcher
	help     widgets.Help

	width  int
	height int
}

func New() (App, error) {
	launcherCfg := widgets.DefaultLauncherConfig()
	clockCfg := widgets.DefaultClockConfig()
	batteryCfg := widgets.DefaultBatteryConfig()
	helpCfg := widgets.DefaultHelpConfig()

	err := Load(&launcherCfg, &clockCfg, &batteryCfg, &helpCfg)

	return App{
		clock:    widgets.NewClock(clockCfg),
		battery:  widgets.NewBattery(batteryCfg),
		launcher: widgets.NewLauncher(launcherCfg),
		help:     widgets.NewHelp(helpCfg),
	}, err
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.clock.Init(), a.battery.Init(), a.launcher.Init())
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+h":
			a.help = a.help.Toggle()
			return a, nil

		case "esc":
			if a.help.Visible() {
				a.help = a.help.Hide()
				return a, nil
			}

			return a, tea.Quit
		}

		if a.help.Visible() {
			return a, nil
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	a.clock, cmd = a.clock.Update(msg)
	cmds = append(cmds, cmd)

	a.battery, cmd = a.battery.Update(msg)
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

	if a.help.Visible() {
		return lipgloss.Place(tuiWidth, tuiHeight, lipgloss.Center, lipgloss.Center, a.help.View())
	}

	left := titleStyle.Render("launtui")

	if a.launcher.Enabled() {
		left = lipgloss.JoinVertical(lipgloss.Left, left, a.launcher.InputView(contentWidth/2))
	}

	header := spread(contentWidth, left, stackRight(a.clock.View(), a.battery.View()))

	divider := dividerStyle.Render(strings.Repeat("─", contentWidth))

	rows := tuiHeight - lipgloss.Height(header) - 1

	sections := []string{header, divider}

	if a.launcher.Enabled() {
		sections = append(sections, a.launcher.ListView(contentWidth, rows))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return appStyle.Width(tuiWidth).Height(tuiHeight).Render(body)
}

func spread(width int, left, right string) string {
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right)
}

func stackRight(parts ...string) string {
	var visible []string

	for _, part := range parts {
		if part != "" {
			visible = append(visible, part)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Right, visible...)
}
