package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newTestApp(t *testing.T, startHotkey string) App {
	t.Helper()

	t.Setenv("LAUNTUI_CONFIG", filepath.Join(t.TempDir(), "config.toml"))

	app, err := New(startHotkey)

	if err != nil {
		t.Fatal(err)
	}

	return app
}

func typeString(model tea.Model, text string) tea.Model {
	for _, r := range text {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	return model
}

func currentName(app App) string {
	return app.modes[app.current].Name()
}

func TestConfigErrorShowsAlertOverlay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	if err := os.WriteFile(path, []byte("[run]\nthis is junk\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LAUNTUI_CONFIG", path)

	app, err := New("")

	if err == nil {
		t.Fatal("a malformed config should report an error")
	}

	if !app.alert.Visible() {
		t.Fatal("a config error should raise the alert overlay")
	}

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if !strings.Contains(model.(App).View(), "error") {
		t.Fatalf("the alert overlay should render on open, got:\n%s", model.(App).View())
	}

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd != nil {
		t.Fatal("esc should dismiss the alert without quitting")
	}

	app = model.(App)

	if app.alert.Visible() {
		t.Fatal("esc should dismiss the alert overlay")
	}

	if !strings.Contains(app.View(), "❯") {
		t.Fatalf("the normal UI should show after dismissal, got:\n%s", app.View())
	}
}

func TestStaleConfigKeyIsIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	if err := os.WriteFile(path, []byte("[run]\nenabled = true\ncomment = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LAUNTUI_CONFIG", path)

	if _, err := New(""); err != nil {
		t.Fatalf("a config carrying a removed key should still load: %v", err)
	}
}

func TestHeaderRow(t *testing.T) {
	cases := []struct {
		name  string
		left  string
		right string
		width int
		want  string
	}{
		{"right aligned with gap", "bar", "clock", 12, "bar    clock"},
		{"min one space gap", "bar", "clock", 9, "bar clock"},
		{"right truncated to fit", "barbarbar", "battery", 14, "barbarbar bat…"},
		{"empty right pads the left", "input", "", 10, "input     "},
	}

	for _, test := range cases {
		if got := headerRow(test.left, test.right, test.width); got != test.want {
			t.Errorf("%s: headerRow(%q, %q, %d) = %q, want %q", test.name, test.left, test.right, test.width, got, test.want)
		}
	}
}

func TestInputWidthReservesOnlyBattery(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	contentWidth := 78
	promptAndCursor := lipgloss.Width(app.input.Prompt) + 1

	want := contentWidth - 0 - 1 - promptAndCursor

	if got := model.(App).input.Width; got != want {
		t.Fatalf("input width = %d, want %d (the full content width less the battery, not the wider clock or half)", got, want)
	}
}

func TestInputExtendsUnderClock(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = typeString(model, "START"+strings.Repeat("-", 56)+"END")

	var inputRow string

	for _, line := range strings.Split(model.(App).View(), "\n") {
		if strings.Contains(line, "❯") {
			inputRow = line
		}
	}

	if inputRow == "" {
		t.Fatal("could not find the search input row")
	}

	if !strings.Contains(inputRow, "START") {
		t.Fatalf("the input scrolled before reaching the status text — the start of the query was hidden:\n%q", inputRow)
	}
}

func TestDefaultIsRunAndAuto(t *testing.T) {
	app := newTestApp(t, "")

	if !app.auto {
		t.Fatal("auto-switching should be on by default")
	}

	if currentName(app) != "Run" {
		t.Fatalf("default mode = %q, want Run", currentName(app))
	}
}

func TestAutoSwitchToCalculator(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = typeString(model, "4+5")

	app = model.(App)

	if currentName(app) != "Calc" {
		t.Fatalf("mode after typing 4+5 = %q, want Calc", currentName(app))
	}

	if !app.auto {
		t.Fatal("auto-switching should stay on")
	}
}

func TestAutoSwitchToWebFallback(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = typeString(model, "how do I update go")

	app = model.(App)

	if currentName(app) != "Web" {
		t.Fatalf("mode after typing a question = %q, want Web", currentName(app))
	}
}

func TestAutoSwitchPrefersWebForURLs(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = typeString(model, "google.com")

	app = model.(App)

	if currentName(app) != "Web" {
		t.Fatalf("mode after typing a URL = %q, want Web", currentName(app))
	}
}

func TestHotkeyDisablesAuto(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	app = model.(App)

	if app.auto {
		t.Fatal("auto-switching should be off after a mode hotkey")
	}

	if currentName(app) != "Calc" {
		t.Fatalf("mode after ctrl+c = %q, want Calc", currentName(app))
	}
}

func TestStartHotkeyOpensMode(t *testing.T) {
	app := newTestApp(t, "ctrl+v")

	if app.auto {
		t.Fatal("starting with a flag should disable auto-switching")
	}

	if currentName(app) != "Clip" {
		t.Fatalf("start mode = %q, want Clip", currentName(app))
	}
}

func TestTabCyclesModes(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	app = model.(App)

	if app.auto {
		t.Fatal("auto-switching should be off after tab")
	}

	if currentName(app) != "Calc" {
		t.Fatalf("mode after tab = %q, want Calc", currentName(app))
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	app = model.(App)

	if currentName(app) != "Run" {
		t.Fatalf("mode after shift+tab = %q, want Run", currentName(app))
	}
}

func TestShiftTabWrapsToLastMode(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	app = model.(App)

	if currentName(app) != "Web" {
		t.Fatalf("mode after shift+tab from the first mode = %q, want Web", currentName(app))
	}
}

func TestCursorNavigatesAndResets(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = typeString(model, "google.com")

	if name := currentName(model.(App)); name != "Web" {
		t.Fatalf("expected Web mode for a URL, got %q", name)
	}

	steps := []struct {
		key        tea.KeyType
		wantCursor int
	}{
		{tea.KeyDown, 1},
		{tea.KeyDown, 1},
		{tea.KeyUp, 0},
	}

	for i, step := range steps {
		model, _ = model.Update(tea.KeyMsg{Type: step.key})

		if cursor := model.(App).cursor; cursor != step.wantCursor {
			t.Fatalf("step %d: cursor = %d, want %d", i, cursor, step.wantCursor)
		}
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = typeString(model, "x")

	if cursor := model.(App).cursor; cursor != 0 {
		t.Fatalf("cursor should reset to 0 when the query changes, got %d", cursor)
	}
}

func TestEscReturnsCloseCommand(t *testing.T) {
	app := newTestApp(t, "")

	var model tea.Model = app
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = typeString(model, "4+5")

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("esc should produce a close command")
	}
}
