package widgets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBattery(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	for name, value := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func batteryView(reading batteryReading) string {
	return Battery{config: BatteryConfig{Enabled: true}, reading: reading}.View()
}

func TestReadBatteryEnergyDischarging(t *testing.T) {
	base := writeBattery(t, map[string]string{
		"status":      "Discharging",
		"capacity":    "50",
		"energy_now":  "30000000",
		"energy_full": "60000000",
		"power_now":   "15000000",
	})

	reading := readBatteryAt(base)

	if !reading.present || reading.percent != 50 || reading.status != "Discharging" {
		t.Fatalf("reading = %+v", reading)
	}

	if reading.hours != 2.0 {
		t.Fatalf("hours = %v, want 2.0", reading.hours)
	}

	view := batteryView(reading)

	if !strings.Contains(view, "2h") {
		t.Fatalf("view = %q, want a 2h estimate", view)
	}

	if !strings.Contains(view, "") {
		t.Fatalf("view = %q, want the half-battery icon", view)
	}
}

func TestReadBatteryChargeCharging(t *testing.T) {
	base := writeBattery(t, map[string]string{
		"status":      "Charging",
		"capacity":    "40",
		"charge_now":  "2000000",
		"charge_full": "5000000",
		"current_now": "1000000",
	})

	reading := readBatteryAt(base)

	if reading.hours != 3.0 {
		t.Fatalf("hours = %v, want 3.0", reading.hours)
	}

	view := batteryView(reading)

	if !strings.Contains(view, "3h") {
		t.Fatalf("view = %q, want a 3h estimate", view)
	}

	if !strings.Contains(view, "") {
		t.Fatalf("view = %q, want the charging bolt icon", view)
	}
}

func TestReadBatteryNoEstimate(t *testing.T) {
	base := writeBattery(t, map[string]string{
		"status":     "Not charging",
		"capacity":   "80",
		"energy_now": "59670000",
		"power_now":  "0",
	})

	reading := readBatteryAt(base)

	if !reading.present || reading.percent != 80 || reading.hours != 0 {
		t.Fatalf("reading = %+v", reading)
	}

	if !strings.Contains(batteryView(reading), "") {
		t.Fatalf("view = %q, want the plug icon", batteryView(reading))
	}
}

func TestReadBatteryAbsent(t *testing.T) {
	if reading := readBatteryAt(writeBattery(t, map[string]string{"present": "0"})); reading.present {
		t.Fatal("present=0 should report absent")
	}

	if reading := readBatteryAt(t.TempDir()); reading.present {
		t.Fatal("empty directory should report absent")
	}
}

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
		view := batteryView(batteryReading{present: true, status: "Discharging", percent: 50, hours: hours})

		if !strings.Contains(view, want) {
			t.Errorf("view for %vh = %q, want it to contain %q", hours, view, want)
		}
	}
}
