package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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
	app, _ := New("")

	if !app.auto {
		t.Fatal("auto-switching should be on by default")
	}

	if currentName(app) != "Run" {
		t.Fatalf("default mode = %q, want Run", currentName(app))
	}
}

func TestAutoSwitchToCalculator(t *testing.T) {
	app, _ := New("")

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

func TestHotkeyDisablesAuto(t *testing.T) {
	app, _ := New("")

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
	app, _ := New("ctrl+v")

	if app.auto {
		t.Fatal("starting with a flag should disable auto-switching")
	}

	if currentName(app) != "Clip" {
		t.Fatalf("start mode = %q, want Clip", currentName(app))
	}
}
