package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/siliconwitch/launtui/internal/widgets"
)

var (
	appStyle          = lipgloss.NewStyle().Padding(0, 1)
	dividerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	modeActiveStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	modeInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type App struct {
	clock   widgets.Clock
	battery widgets.Battery
	help    widgets.Help

	input   textinput.Model
	modes   []widgets.Mode
	current int
	auto    bool

	width  int
	height int
}

func New(startHotkey string) (App, error) {
	runCfg := widgets.DefaultRunConfig()
	calculatorCfg := widgets.DefaultCalculatorConfig()
	passwordsCfg := widgets.DefaultPasswordsConfig()
	projectsCfg := widgets.DefaultProjectsConfig()
	clipboardCfg := widgets.DefaultClipboardConfig()
	webCfg := widgets.DefaultWebConfig()
	clockCfg := widgets.DefaultClockConfig()
	batteryCfg := widgets.DefaultBatteryConfig()
	helpCfg := widgets.DefaultHelpConfig()

	err := Load(&runCfg, &calculatorCfg, &passwordsCfg, &projectsCfg, &clipboardCfg,
		&webCfg, &clockCfg, &batteryCfg, &helpCfg)

	input := textinput.New()
	input.Prompt = "❯ "
	input.Placeholder = "Search…"
	input.Focus()

	app := App{
		clock:   widgets.NewClock(clockCfg),
		battery: widgets.NewBattery(batteryCfg),
		input:   input,
		modes: []widgets.Mode{
			widgets.NewRun(runCfg),
			widgets.NewCalculator(calculatorCfg),
			widgets.NewPasswords(passwordsCfg),
			widgets.NewProjects(projectsCfg),
			widgets.NewClipboard(clipboardCfg),
			widgets.NewWeb(webCfg),
		},
		auto: true,
	}

	app.current = app.defaultMode()

	if startHotkey != "" {
		for i, mode := range app.modes {
			if mode.Enabled() && mode.Hotkey() == startHotkey {
				app.current = i
				app.auto = false
			}
		}
	}

	app.help = widgets.NewHelp(helpCfg).WithBindings(app.helpBindings())

	return app, err
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, a.clock.Init(), a.battery.Init()}

	for _, mode := range a.modes {
		cmds = append(cmds, mode.Init())
	}

	return tea.Batch(cmds...)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.input.Width = a.inputWidth()

		return a, nil

	case widgets.RequestQuitMsg:
		cmd := a.close()

		return a, cmd

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	if isCtrlDelete(msg) {
		return a.clearCurrentHistory()
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	previous := a.input.Value()

	a.input, cmd = a.input.Update(msg)
	cmds = append(cmds, cmd)

	a.clock, cmd = a.clock.Update(msg)
	cmds = append(cmds, cmd)

	a.battery, cmd = a.battery.Update(msg)
	cmds = append(cmds, cmd)

	for i := range a.modes {
		a.modes[i], cmd = a.modes[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	if a.input.Value() != previous {
		a.setQuery(a.input.Value())
	}

	if a.auto {
		a.autoSwitch()
	}

	return a, tea.Batch(cmds...)
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+h" {
		a.help = a.help.Toggle()

		return a, nil
	}

	if a.help.Visible() {
		if key == "esc" {
			a.help = a.help.Hide()
		}

		return a, nil
	}

	if key == "esc" {
		cmd := a.close()

		return a, cmd
	}

	for i, mode := range a.modes {
		if mode.Enabled() && mode.Hotkey() == key {
			a.current = i
			a.auto = false

			return a, nil
		}
	}

	switch key {
	case "tab":
		a.current = a.adjacentMode(1)
		a.auto = false

		return a, nil

	case "shift+tab":
		a.current = a.adjacentMode(-1)
		a.auto = false

		return a, nil

	case "up":
		a.modes[a.current] = a.modes[a.current].MoveUp()

		return a, nil

	case "down":
		a.modes[a.current] = a.modes[a.current].MoveDown()

		return a, nil

	case "enter":
		return a, a.modes[a.current].Activate()

	case "delete":
		if editor, ok := a.modes[a.current].(widgets.HistoryEditor); ok {
			if mode, cmd, handled := editor.DeleteSelectedHistory(); handled {
				a.modes[a.current] = mode

				return a, cmd
			}
		}
	}

	previous := a.input.Value()

	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)

	if a.input.Value() != previous {
		a.setQuery(a.input.Value())

		if a.auto {
			a.autoSwitch()
		}
	}

	return a, cmd
}

func (a App) adjacentMode(delta int) int {
	count := len(a.modes)

	for step := 1; step <= count; step++ {
		index := ((a.current+delta*step)%count + count) % count

		if a.modes[index].Enabled() {
			return index
		}
	}

	return a.current
}

var ctrlDeleteSequences = map[string]bool{
	unknownCSIString("3;5~"): true,
	unknownCSIString("3^"):   true,
}

func unknownCSIString(parameters string) string {
	return fmt.Sprintf("?CSI%+v?", []byte(parameters))
}

func isCtrlDelete(msg tea.Msg) bool {
	sequence, ok := msg.(fmt.Stringer)

	return ok && ctrlDeleteSequences[sequence.String()]
}

func (a App) clearCurrentHistory() (tea.Model, tea.Cmd) {
	if a.help.Visible() {
		return a, nil
	}

	if editor, ok := a.modes[a.current].(widgets.HistoryEditor); ok {
		mode, cmd := editor.ClearHistory()
		a.modes[a.current] = mode

		return a, cmd
	}

	return a, nil
}

func (a *App) setQuery(query string) {
	for i := range a.modes {
		if a.modes[i].Enabled() {
			a.modes[i] = a.modes[i].SetQuery(query)
		}
	}
}

func (a *App) close() tea.Cmd {
	var cmds []tea.Cmd

	for i := range a.modes {
		if !a.modes[i].Enabled() {
			continue
		}

		var cmd tea.Cmd

		a.modes[i], cmd = a.modes[i].Update(widgets.AppClosingMsg{})

		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) == 0 {
		return tea.Quit
	}

	return tea.Sequence(tea.Batch(cmds...), tea.Quit)
}

func (a *App) autoSwitch() {
	for i, mode := range a.modes {
		if !mode.Enabled() {
			continue
		}

		if strong, ok := mode.(widgets.StrongMatcher); ok && strong.StrongMatch() {
			a.current = i

			return
		}
	}

	for i, mode := range a.modes {
		if mode.Enabled() && mode.HasResults() {
			a.current = i

			return
		}
	}

	a.current = a.defaultMode()
}

func (a App) defaultMode() int {
	for i, mode := range a.modes {
		if mode.Enabled() {
			return i
		}
	}

	return 0
}

func (a App) helpBindings() []widgets.HelpBinding {
	bindings := []widgets.HelpBinding{
		{Keys: "type", Desc: "filter the list"},
		{Keys: "↑ / ↓", Desc: "move selection"},
		{Keys: "enter", Desc: "activate selection"},
		{Keys: "tab / shift+tab", Desc: "next / previous mode"},
		{Keys: "del", Desc: "delete the selected history entry"},
		{Keys: "ctrl+del", Desc: "clear the mode's history"},
	}

	for _, mode := range a.modes {
		if mode.Enabled() {
			bindings = append(bindings, widgets.HelpBinding{Keys: mode.Hotkey(), Desc: mode.Name() + " mode"})
		}
	}

	return append(bindings,
		widgets.HelpBinding{Keys: "ctrl+h", Desc: "toggle this help"},
		widgets.HelpBinding{Keys: "esc", Desc: "quit"},
	)
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

	left := lipgloss.JoinVertical(lipgloss.Left,
		a.modeBar(),
		a.input.View(),
	)

	header := spread(contentWidth, left, stackRight(a.clock.View(), a.battery.View()))

	divider := dividerStyle.Render(strings.Repeat("─", contentWidth))

	rows := tuiHeight - lipgloss.Height(header) - 1

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		divider,
		a.modes[a.current].View(contentWidth, rows),
	)

	return appStyle.Width(tuiWidth).Height(tuiHeight).Render(body)
}

func (a App) inputWidth() int {
	contentWidth := max(1, a.width-2)

	return max(1, contentWidth/2-lipgloss.Width(a.input.Prompt)-1)
}

func (a App) modeBar() string {
	var parts []string

	for i, mode := range a.modes {
		if !mode.Enabled() {
			continue
		}

		if i == a.current {
			parts = append(parts, modeActiveStyle.Render(mode.Name()))
		} else {
			parts = append(parts, modeInactiveStyle.Render(mode.Name()))
		}
	}

	bar := strings.Join(parts, "  ")

	if a.auto {
		bar += modeInactiveStyle.Render("  · auto")
	}

	return bar
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
