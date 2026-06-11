//go:build !darwin

package widgets

import (
	"os"
	"path/filepath"
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

	if got := batteryDuration(reading.hours); got != "2h" {
		t.Fatalf("duration = %q, want %q", got, "2h")
	}

	if got := batteryIcon(reading); got != "" {
		t.Fatalf("icon = %q, want half-battery", got)
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

	if got := batteryDuration(reading.hours); got != "3h" {
		t.Fatalf("duration = %q, want %q", got, "3h")
	}

	if got := batteryIcon(reading); got != "" {
		t.Fatalf("icon = %q, want bolt", got)
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

	if got := batteryIcon(reading); got != "" {
		t.Fatalf("icon = %q, want plug", got)
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
