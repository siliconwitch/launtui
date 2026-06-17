package widgets

import (
	"strings"
	"testing"
)

func TestAlert(t *testing.T) {
	if NewAlert("").Visible() {
		t.Fatal("an empty message should not raise the alert")
	}

	alert := NewAlert("config section [run]: bad value")

	if !alert.Visible() {
		t.Fatal("a non-empty message should raise the alert")
	}

	if !strings.Contains(alert.View(), "config section [run]: bad value") {
		t.Fatalf("the message should appear in the view: %q", alert.View())
	}

	if alert.Hide().Visible() {
		t.Fatal("Hide should clear visibility")
	}
}
