package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

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
	alert   widgets.Alert

	input   textinput.Model
	modes   []widgets.Mode
	current int
	cursor  int
	auto    bool

	draft     string
	recalling bool

	width  int
	height int
}

func New(startHotkey string) (App, error) {
	runConfig := widgets.DefaultRunConfig()
	calculatorConfig := widgets.DefaultCalculatorConfig()
	passwordsConfig := widgets.DefaultPasswordsConfig()
	projectsConfig := widgets.DefaultProjectsConfig()
	clipboardConfig := widgets.DefaultClipboardConfig()
	emojiConfig := widgets.DefaultEmojiConfig()
	webConfig := widgets.DefaultWebConfig()
	clockConfig := widgets.DefaultClockConfig()
	batteryConfig := widgets.DefaultBatteryConfig()
	helpConfig := widgets.DefaultHelpConfig()

	err := LoadConfig(
		&runConfig,
		&calculatorConfig,
		&passwordsConfig,
		&projectsConfig,
		&clipboardConfig,
		&emojiConfig,
		&webConfig,
		&clockConfig,
		&batteryConfig,
		&helpConfig,
	)

	input := textinput.New()
	input.Prompt = "❯ "
	input.Focus()

	app := App{
		clock:   widgets.NewClock(clockConfig),
		battery: widgets.NewBattery(batteryConfig),
		input:   input,
		modes: []widgets.Mode{
			widgets.NewRun(runConfig),
			widgets.NewCalculator(calculatorConfig),
			widgets.NewPasswords(passwordsConfig),
			widgets.NewProjects(projectsConfig),
			widgets.NewClipboard(clipboardConfig),
			widgets.NewEmoji(emojiConfig),
			widgets.NewWeb(webConfig),
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

	bindings := []widgets.HelpBinding{
		{Keys: "type", Description: "filter the list"},
		{Keys: "↑ / ↓", Description: "move selection"},
		{Keys: "enter", Description: "activate selection"},
		{Keys: "esc", Description: "quit"},
		{Keys: "tab / shift+tab", Description: "next / previous mode"},
		{Keys: "del", Description: "delete the selected history entry"},
		{Keys: "alt+del", Description: "clear the mode's history"},
	}

	if len(clockConfig.Zones) > 0 {
		bindings = append(bindings, widgets.HelpBinding{Keys: "ctrl+t", Description: "switch time zone"})
	}

	for _, mode := range app.modes {
		if mode.Enabled() {
			bindings = append(bindings, widgets.HelpBinding{Keys: mode.Hotkey(), Description: mode.Name() + " mode"})
		}
	}

	app.help = widgets.NewHelp(helpConfig).WithBindings(bindings)

	if err != nil {
		app.alert = widgets.NewAlert(err.Error())
	}

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

		a.resizeInput()

		return a, nil

	case widgets.RequestQuitMsg:
		return a, a.close()

	case tea.KeyMsg:
		key := msg.String()

		if a.alert.Visible() {
			if key == "esc" {
				a.alert = a.alert.Hide()
			}

			return a, nil
		}

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
			return a, a.close()
		}

		for i, mode := range a.modes {
			if mode.Enabled() && mode.Hotkey() == key {
				a.current = i
				a.auto = false
				a.cursor = 0
				a.recalling = false

				return a, a.notifySelection()
			}
		}

		var cmd tea.Cmd

		switch key {
		case "tab":
			a.current = a.adjacentMode(1)
			a.auto = false
			a.cursor = 0
			a.recalling = false

			return a, a.notifySelection()

		case "shift+tab":
			a.current = a.adjacentMode(-1)
			a.auto = false
			a.cursor = 0
			a.recalling = false

			return a, a.notifySelection()

		case "ctrl+t":
			a.clock = a.clock.NextZone()

			return a, nil

		case "up":
			a.navigate(-1)

			return a, a.notifySelection()

		case "down":
			a.navigate(1)

			return a, a.notifySelection()

		case "enter":
			return a, a.modes[a.current].Activate(a.cursor)

		case "delete":
			if deleter, ok := a.modes[a.current].(widgets.RowDeleter); ok {
				rows := a.modes[a.current].Rows()

				if a.cursor < len(rows) && rows[a.cursor].Deletable {
					a.modes[a.current], cmd = deleter.DeleteRow(a.cursor)
					a.clampCursor()

					return a, cmd
				}
			}

			return a, nil

		case "alt+delete":
			if deleter, ok := a.modes[a.current].(widgets.RowDeleter); ok {
				a.modes[a.current], cmd = deleter.ClearRows()
				a.clampCursor()

				return a, cmd
			}

			return a, nil
		}

		previous := a.input.Value()

		a.input, cmd = a.input.Update(msg)

		if a.input.Value() != previous {
			a.setQuery(a.input.Value())

			if a.auto {
				a.autoSwitch()
			}

			cmd = tea.Batch(cmd, a.notifySelection())
		}

		return a, cmd
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

	a.resizeInput()

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

func (a *App) clampCursor() {
	if rows := len(a.modes[a.current].Rows()); a.cursor >= rows {
		a.cursor = max(rows-1, 0)
	}
}

func (a *App) notifySelection() tea.Cmd {
	selectable, ok := a.modes[a.current].(widgets.Selectable)

	if !ok {
		return nil
	}

	var cmd tea.Cmd

	a.modes[a.current], cmd = selectable.Select(a.cursor)

	return cmd
}

func (a *App) setQuery(query string) {
	a.cursor = 0
	a.recalling = false

	a.applyQuery(query)
}

func (a *App) applyQuery(query string) {
	for i := range a.modes {
		if a.modes[i].Enabled() {
			a.modes[i] = a.modes[i].SetQuery(query)
		}
	}
}

func (a *App) navigate(delta int) {
	recaller, isRecaller := a.modes[a.current].(widgets.Recaller)

	move := true

	if isRecaller && !a.recalling {
		if _, onHistory := recaller.RecallText(a.cursor); onHistory {
			move = false
		}
	}

	if move {
		a.cursor += delta
	}

	if a.cursor < 0 {
		a.cursor = 0
	}

	a.clampCursor()

	if isRecaller {
		a.recall(recaller)
	}
}

func (a *App) recall(recaller widgets.Recaller) {
	text, onHistory := recaller.RecallText(a.cursor)

	switch {
	case onHistory:
		if !a.recalling {
			a.draft = a.input.Value()
			a.recalling = true
		}
	case a.recalling:
		text = a.draft
		a.recalling = false
	default:
		return
	}

	a.input.SetValue(text)
	a.input.CursorEnd()

	before := len(a.modes[a.current].Rows())

	a.applyQuery(text)

	a.cursor += len(a.modes[a.current].Rows()) - before

	if a.cursor < 0 {
		a.cursor = 0
	}

	a.clampCursor()
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
		if mode.Enabled() && widgets.HasResults(mode.Rows()) {
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

func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}

	tuiWidth := max(1, a.width)
	tuiHeight := max(1, a.height)
	contentWidth := max(1, a.width-2)

	if a.alert.Visible() {
		return lipgloss.Place(tuiWidth, tuiHeight, lipgloss.Center, lipgloss.Center, a.alert.View())
	}

	if a.help.Visible() {
		return lipgloss.Place(tuiWidth, tuiHeight, lipgloss.Center, lipgloss.Center, a.help.View())
	}

	var modes []string

	for i, mode := range a.modes {
		if !mode.Enabled() {
			continue
		}

		if i == a.current {
			modes = append(modes, modeActiveStyle.Render(mode.Name()))
		} else {
			modes = append(modes, modeInactiveStyle.Render(mode.Name()))
		}
	}

	bar := strings.Join(modes, "  ")

	a.input.Placeholder = "Search or ctrl+h for help"

	if a.auto {
		a.input.Placeholder += " (auto mode)"
	}

	clock := a.clock.View()
	battery := a.battery.View()

	header := lipgloss.JoinVertical(lipgloss.Left,
		headerRow(bar, clock, contentWidth),
		headerRow(a.input.View(), battery, contentWidth),
	)

	divider := dividerStyle.Render(strings.Repeat("─", contentWidth))

	bodyHeight := tuiHeight - lipgloss.Height(header) - 1

	mode := a.modes[a.current]
	rows := mode.Rows()
	cursor := a.cursor

	if cursor >= len(rows) {
		cursor = max(len(rows)-1, 0)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		divider,
		widgets.RenderResults(mode.Status(), rows, mode.Accent(), cursor, contentWidth, bodyHeight),
	)

	return appStyle.Width(tuiWidth).Height(tuiHeight).Render(body)
}

func (a *App) resizeInput() {
	contentWidth := max(1, a.width-2)
	batteryWidth := lipgloss.Width(a.battery.View())
	promptAndCursor := lipgloss.Width(a.input.Prompt) + 1

	a.input.Width = max(1, contentWidth-batteryWidth-1-promptAndCursor)
}

func headerRow(left, right string, width int) string {
	leftWidth := lipgloss.Width(left)

	if leftWidth+lipgloss.Width(right)+1 > width {
		right = ansi.Truncate(right, max(0, width-leftWidth-1), "…")
	}

	gap := max(1, width-leftWidth-lipgloss.Width(right))

	return left + strings.Repeat(" ", gap) + right
}

type Section interface {
	SectionName() string
}

func LoadConfig(targets ...Section) error {
	path := os.Getenv("LAUNTUI_CONFIG")

	if path == "" {
		configDir, err := os.UserConfigDir()

		if err != nil {
			return err
		}

		path = filepath.Join(configDir, "launtui", "config.toml")
	}

	var raw map[string]toml.Primitive

	metadata, err := toml.DecodeFile(path, &raw)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("reading %s: %w", path, err)
	}

	for _, target := range targets {
		primitive, ok := raw[target.SectionName()]

		if !ok {
			continue
		}

		if err := metadata.PrimitiveDecode(primitive, target); err != nil {
			return fmt.Errorf("config section [%s]: %w", target.SectionName(), err)
		}
	}

	return nil
}
