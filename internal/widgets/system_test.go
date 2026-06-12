package widgets

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

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

	suppressClipboardRecording("hunter2")

	if !clipboardRecordingSuppressed("hunter2") {
		t.Fatal("suppressed text should be recognised")
	}

	if clipboardRecordingSuppressed("other text") {
		t.Fatal("other text should not be suppressed")
	}

	hidden := recordClipboardText("hunter2", 5)

	if len(hidden) != 0 {
		t.Fatalf("suppressed text should not be recorded, got %+v", hidden)
	}

	recorded := recordClipboardText("user@example.com", 5)

	if len(recorded) != 1 || recorded[0].Text != "user@example.com" {
		t.Fatalf("normal text should still be recorded, got %+v", recorded)
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
