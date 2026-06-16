package widgets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestClockZoneCycle(t *testing.T) {
	config := DefaultClockConfig()
	config.Zones = []string{"Europe/London", "America/Los_Angeles", "bogus/zone"}

	clock := NewClock(config)

	if len(clock.zones) != 3 {
		t.Fatalf("zones = %d, want 3 (local + 2 valid, bogus dropped)", len(clock.zones))
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

func TestClockCityNames(t *testing.T) {
	config := DefaultClockConfig()
	config.Zones = []string{"San Francisco", "tokyo"}

	clock := NewClock(config)

	if len(clock.zones) != 3 {
		t.Fatalf("zones = %d, want 3 (local + 2 cities)", len(clock.zones))
	}

	sf := clock.NextZone()

	if got := ansi.Strip(sf.View()); !strings.Contains(got, "(San Francisco)") {
		t.Fatalf("expected pinned city '(San Francisco)' in view, got %q", got)
	}

	if zone := clock.zones[1].location.String(); zone != "America/Los_Angeles" {
		t.Fatalf("San Francisco should resolve to America/Los_Angeles, got %q", zone)
	}

	tokyo := sf.NextZone()

	if got := ansi.Strip(tokyo.View()); !strings.Contains(got, "(Tokyo)") {
		t.Fatalf("expected '(Tokyo)' (lowercase input title-cased), got %q", got)
	}
}
