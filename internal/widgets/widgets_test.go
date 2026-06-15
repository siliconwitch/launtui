package widgets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestRenderResults(t *testing.T) {
	rows := []Row{
		{left: "alpha", right: "9m"},
		{left: "beta"},
		{left: "gamma", style: rowDim, Deletable: true},
	}

	accent := lipgloss.Color("4")

	lines := strings.Split(ansi.Strip(RenderResults("", rows, accent, 1, 40, 10)), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 rows, got %d: %q", len(lines), lines)
	}

	if !strings.HasPrefix(lines[1], "▌ ") {
		t.Fatalf("selected row should show the cursor marker, got %q", lines[1])
	}

	if strings.HasPrefix(lines[0], "▌") {
		t.Fatalf("unselected row should not show the cursor marker, got %q", lines[0])
	}

	if !strings.HasSuffix(lines[0], "9m") {
		t.Fatalf("right content should be right-aligned, got %q", lines[0])
	}

	withStatus := strings.Split(ansi.Strip(RenderResults("loading…", rows, accent, 0, 40, 10)), "\n")

	if len(withStatus) != 4 || withStatus[0] != "loading…" {
		t.Fatalf("status should render above the rows, got %q", withStatus)
	}

	windowed := strings.Split(ansi.Strip(RenderResults("", rows, accent, 2, 40, 2)), "\n")

	if len(windowed) != 2 || !strings.Contains(windowed[1], "gamma") {
		t.Fatalf("windowing should keep the cursor visible, got %q", windowed)
	}
}

func TestRelativeAge(t *testing.T) {
	cases := map[int64]string{
		30:    "now",
		90:    "1m",
		3600:  "1h",
		90000: "1d",
	}

	for elapsed, want := range cases {
		if got := relativeAge(elapsed); got != want {
			t.Errorf("relativeAge(%d) = %q, want %q", elapsed, got, want)
		}
	}
}

func TestHasResults(t *testing.T) {
	if HasResults(nil) {
		t.Fatal("no rows should not count as results")
	}

	if HasResults([]Row{{left: "history", style: rowDim}}) {
		t.Fatal("only dimmed history rows should not count as results")
	}

	if !HasResults([]Row{{left: "match"}, {left: "history", style: rowDim}}) {
		t.Fatal("a non-dim row should count as a result")
	}
}
