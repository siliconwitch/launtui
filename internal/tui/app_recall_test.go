package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func recallApp(t *testing.T) App {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("LAUNTUI_CONFIG", filepath.Join(dir, "config.toml"))
	t.Setenv("XDG_DATA_HOME", dir)

	store := filepath.Join(dir, "launtui")

	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}

	history := `[
		{"expression":"9*9","answer":"81","time":1000},
		{"expression":"2+2","answer":"4","time":900},
		{"expression":"5+5","answer":"10","time":800}
	]`

	if err := os.WriteFile(filepath.Join(store, "calculator-history.json"), []byte(history), 0o644); err != nil {
		t.Fatal(err)
	}

	app, err := New("ctrl+c")
	if err != nil {
		t.Fatal(err)
	}

	var model tea.Model = app

	for _, mode := range app.modes {
		if mode.Name() != "Calc" {
			continue
		}

		batch, ok := mode.Init()().(tea.BatchMsg)

		if !ok {
			t.Fatal("calculator Init did not return a batch")
		}

		model, _ = model.Update(batch[0]())
	}

	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	return model.(App)
}

func press(model tea.Model, key tea.KeyType) tea.Model {
	model, _ = model.Update(tea.KeyMsg{Type: key})

	return model
}

func TestRecallFromTypedDraft(t *testing.T) {
	var model tea.Model = recallApp(t)
	model = typeString(model, "3*3")

	steps := []struct {
		key    tea.KeyType
		input  string
		cursor int
	}{
		{tea.KeyDown, "9*9", 1},
		{tea.KeyDown, "2+2", 2},
		{tea.KeyUp, "9*9", 1},
		{tea.KeyUp, "3*3", 0},
	}

	for i, step := range steps {
		model = press(model, step.key)
		app := model.(App)

		if app.input.Value() != step.input || app.cursor != step.cursor {
			t.Errorf("step %d: input=%q cursor=%d, want %q %d", i, app.input.Value(), app.cursor, step.input, step.cursor)
		}
	}
}

func TestRecallFromEmptyDraft(t *testing.T) {
	var model tea.Model = recallApp(t)

	steps := []struct {
		key    tea.KeyType
		input  string
		cursor int
	}{
		{tea.KeyDown, "9*9", 1},
		{tea.KeyDown, "2+2", 2},
		{tea.KeyUp, "9*9", 1},
		{tea.KeyUp, "", 0},
	}

	for i, step := range steps {
		model = press(model, step.key)
		app := model.(App)

		if app.input.Value() != step.input || app.cursor != step.cursor {
			t.Errorf("step %d: input=%q cursor=%d, want %q %d", i, app.input.Value(), app.cursor, step.input, step.cursor)
		}
	}
}

func webRecallApp(t *testing.T) App {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("LAUNTUI_CONFIG", filepath.Join(dir, "config.toml"))
	t.Setenv("XDG_DATA_HOME", dir)

	store := filepath.Join(dir, "launtui")

	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}

	history := `[
		{"label":"Search the web for “golang”","url":"https://duckduckgo.com/?q=golang","query":"golang","time":1000},
		{"label":"Open https://github.com","url":"https://github.com","query":"github.com","time":900}
	]`

	if err := os.WriteFile(filepath.Join(store, "web-history.json"), []byte(history), 0o644); err != nil {
		t.Fatal(err)
	}

	app, err := New("ctrl+s")
	if err != nil {
		t.Fatal(err)
	}

	var model tea.Model = app

	for _, mode := range app.modes {
		if mode.Name() == "Web" {
			model, _ = model.Update(mode.Init()())
		}
	}

	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	return model.(App)
}

func TestWebRecallFromTypedDraft(t *testing.T) {
	var model tea.Model = webRecallApp(t)
	model = typeString(model, "rust")

	steps := []struct {
		key    tea.KeyType
		input  string
		cursor int
	}{
		{tea.KeyDown, "golang", 1},
		{tea.KeyDown, "github.com", 3},
		{tea.KeyUp, "golang", 1},
		{tea.KeyUp, "rust", 0},
	}

	for i, step := range steps {
		model = press(model, step.key)
		app := model.(App)

		if app.input.Value() != step.input || app.cursor != step.cursor {
			t.Errorf("step %d: input=%q cursor=%d, want %q %d", i, app.input.Value(), app.cursor, step.input, step.cursor)
		}
	}
}

func TestRecallEndsOnEdit(t *testing.T) {
	var model tea.Model = recallApp(t)
	model = typeString(model, "3*3")
	model = press(model, tea.KeyDown)

	if v := model.(App).input.Value(); v != "9*9" {
		t.Fatalf("after down, input = %q, want 9*9", v)
	}

	model = typeString(model, "0")
	app := model.(App)

	if app.input.Value() != "9*90" || app.recalling || app.cursor != 0 {
		t.Fatalf("after edit: input=%q recalling=%v cursor=%d", app.input.Value(), app.recalling, app.cursor)
	}
}
