package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
