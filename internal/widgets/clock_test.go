package widgets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestClockZoneCycle(t *testing.T) {
	cfg := DefaultClockConfig()
	cfg.Zones = []string{"Europe/London", "America/Los_Angeles", "bogus/zone"}

	clock := NewClock(cfg)

	if len(clock.locations) != 3 {
		t.Fatalf("locations = %d, want 3 (local + 2 valid, bogus dropped)", len(clock.locations))
	}

	if clock.current != 0 {
		t.Fatal("clock should start at local time")
	}

	if strings.Contains(ansi.Strip(clock.View()), "London") {
		t.Fatal("local view should not show a zone label")
	}

	london := clock.NextZone()

	if london.current != 1 || !strings.Contains(ansi.Strip(london.View()), "London") {
		t.Fatalf("after one toggle, expected London label: current=%d view=%q", london.current, london.View())
	}

	la := london.NextZone()

	if !strings.Contains(ansi.Strip(la.View()), "Los Angeles") {
		t.Fatalf("expected 'Los Angeles' label (underscore replaced), got %q", la.View())
	}

	if wrapped := la.NextZone(); wrapped.current != 0 {
		t.Fatalf("toggle should wrap back to local, got current=%d", wrapped.current)
	}
}
