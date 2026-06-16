package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func dividerRow(view string) int {
	for i, line := range strings.Split(view, "\n") {
		trimmed := strings.TrimSpace(ansi.Strip(line))

		if trimmed != "" && strings.Trim(trimmed, "─") == "" {
			return i
		}
	}

	return -1
}

func TestHeaderStaysTwoRowsWithFilledInput(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(cfg, []byte("[clock]\nzones = [\"San Francisco\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LAUNTUI_CONFIG", cfg)

	for _, width := range []int{70, 80, 100} {
		app, err := New("")

		if err != nil {
			t.Fatal(err)
		}

		var model tea.Model = app
		model, _ = model.Update(tea.WindowSizeMsg{Width: width, Height: 12})
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
		model = typeString(model, "1234 kilometers to nautical miles")

		if row := dividerRow(model.(App).View()); row != 2 {
			t.Errorf("width=%d: divider on row %d, want 2 (header wrapped onto an extra line)", width, row)
		}
	}
}
