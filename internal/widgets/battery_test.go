package widgets

import "testing"

func TestBatteryDuration(t *testing.T) {
	cases := map[float64]string{
		3.0:  "3h",
		3.2:  "3.2h",
		2.0:  "2h",
		1.5:  "90m",
		1.25: "75m",
		0.5:  "30m",
	}

	for hours, want := range cases {
		if got := batteryDuration(hours); got != want {
			t.Errorf("batteryDuration(%v) = %q, want %q", hours, got, want)
		}
	}
}
