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

func TestRenderRowReservesRightColumn(t *testing.T) {
	accent := lipgloss.Color("4")

	cases := []struct {
		name     string
		left     string
		right    string
		width    int
		endsWith string
		mustShow string
	}{
		{name: "short left keeps full right", left: "alpha", right: "9m", width: 40, endsWith: "9m", mustShow: "alpha"},
		{name: "long left truncates before its right", left: strings.Repeat("x", 100), right: "9m", width: 20, endsWith: "9m"},
		{name: "no right uses full width", left: strings.Repeat("y", 100), right: "", width: 20},
		{name: "narrow width still right-aligns", left: "some entry text here", right: "12h", width: 16, endsWith: "12h"},
		{name: "long right is capped, primary survives", left: "personal-email-work-account", right: strings.Repeat("z", 46), width: 50, mustShow: "personal-email-work"},
	}

	for _, test := range cases {
		rendered := ansi.Strip(renderRow(accent, false, Row{left: test.left, right: test.right}, test.width))

		if lipgloss.Width(rendered) > test.width {
			t.Errorf("%s: width %d exceeds %d: %q", test.name, lipgloss.Width(rendered), test.width, rendered)
		}

		if test.endsWith != "" && !strings.HasSuffix(rendered, test.endsWith) {
			t.Errorf("%s: right column should be preserved at the edge, got %q", test.name, rendered)
		}

		if test.mustShow != "" && !strings.Contains(rendered, test.mustShow) {
			t.Errorf("%s: primary text %q should survive, got %q", test.name, test.mustShow, rendered)
		}
	}
}

func TestRenderRowNeverExceedsWidth(t *testing.T) {
	accent := lipgloss.Color("4")

	rights := []string{"", "9m", strings.Repeat("z", 30)}

	for width := 1; width <= 24; width++ {
		for _, right := range rights {
			for _, selected := range []bool{false, true} {
				rendered := renderRow(accent, selected, Row{left: strings.Repeat("a", 40), right: right}, width)

				if got := lipgloss.Width(ansi.Strip(rendered)); got > width {
					t.Errorf("width=%d right=%q selected=%v: rendered width %d exceeds it", width, right, selected, got)
				}
			}
		}
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

func TestTruncateByDisplayWidth(t *testing.T) {
	if got := truncate("hello", 4); got != "hel…" {
		t.Fatalf("truncate(hello, 4) = %q", got)
	}

	if got := truncate("héllo", 10); got != "héllo" {
		t.Fatalf("truncate(héllo, 10) = %q", got)
	}

	wide := truncate("日本語テキスト", 5)

	if lipgloss.Width(wide) > 5 {
		t.Fatalf("truncate(wide, 5) = %q (width %d)", wide, lipgloss.Width(wide))
	}
}

func TestTerminalArgv(t *testing.T) {
	cases := map[string][]string{
		"foot":           {"foot", "sh", "-c", "btop"},
		"kitty":          {"kitty", "sh", "-c", "btop"},
		"alacritty":      {"alacritty", "-e", "sh", "-c", "btop"},
		"wezterm":        {"wezterm", "start", "--", "sh", "-c", "btop"},
		"gnome-terminal": {"gnome-terminal", "--", "sh", "-c", "btop"},
		"":               {"sh", "-c", "btop"},
	}

	for terminal, want := range cases {
		got := terminalArgv(terminal, "btop")

		if len(got) != len(want) {
			t.Errorf("terminalArgv(%q) = %v, want %v", terminal, got, want)
			continue
		}

		for i := range want {
			if got[i] != want[i] {
				t.Errorf("terminalArgv(%q) = %v, want %v", terminal, got, want)
				break
			}
		}
	}
}

func TestRecordClipboardText(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	first := recordClipboardText("alpha", 3)

	if len(first) != 1 || first[0].Text != "alpha" {
		t.Fatalf("first record = %+v", first)
	}

	recordClipboardText("beta", 3)

	moved := recordClipboardText("alpha", 3)

	if len(moved) != 2 || moved[0].Text != "alpha" || moved[1].Text != "beta" {
		t.Fatalf("re-recording should move to front, got %+v", moved)
	}

	recordClipboardText("gamma", 3)

	capped := recordClipboardText("delta", 3)

	if len(capped) != 3 || capped[0].Text != "delta" {
		t.Fatalf("capped history = %+v", capped)
	}

	blank := recordClipboardText("   ", 3)

	if len(blank) != 3 {
		t.Fatalf("blank text should not be recorded, got %+v", blank)
	}

	untrimmed := recordClipboardText("epsilon", 0)

	if len(untrimmed) != 4 {
		t.Fatalf("limit 0 should not trim existing entries, got %+v", untrimmed)
	}
}

func TestClipboardSuppression(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	if err := suppressClipboardRecording("hunter2"); err != nil {
		t.Fatal(err)
	}

	if hidden := recordClipboardText("hunter2", 5); len(hidden) != 0 {
		t.Fatalf("suppressed text should not be recorded, got %+v", hidden)
	}

	recorded := recordClipboardText("user@example.com", 5)

	if len(recorded) != 1 || recorded[0].Text != "user@example.com" {
		t.Fatalf("non-suppressed text should still be recorded, got %+v", recorded)
	}
}
