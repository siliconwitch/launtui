//go:build !darwin

package widgets

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readBattery(device string) batteryReading {
	reading := readBatteryAt(filepath.Join("/sys/class/power_supply", device))

	if reading.present {
		return reading
	}

	if base := firstBatteryDevice(); base != "" {
		return readBatteryAt(base)
	}

	return reading
}

func firstBatteryDevice() string {
	entries, err := os.ReadDir("/sys/class/power_supply")

	if err != nil {
		return ""
	}

	for _, entry := range entries {
		base := filepath.Join("/sys/class/power_supply", entry.Name())

		if readSysString(base, "type") == "Battery" {
			return base
		}
	}

	return ""
}

func readBatteryAt(base string) batteryReading {
	if readSysString(base, "present") == "0" {
		return batteryReading{}
	}

	status := readSysString(base, "status")
	capacity, hasCapacity := readSysInt(base, "capacity")

	if status == "" && !hasCapacity {
		return batteryReading{}
	}

	return batteryReading{
		present: true,
		status:  status,
		percent: int(capacity),
		hours:   batteryHours(base, status),
	}
}

func batteryHours(base, status string) float64 {
	now, full, rate, ok := batteryCharge(base)

	if !ok || rate <= 0 {
		return 0
	}

	switch status {
	case "Discharging":
		return float64(now) / float64(rate)
	case "Charging":
		if full > now {
			return float64(full-now) / float64(rate)
		}
	}

	return 0
}

func batteryCharge(base string) (now, full, rate int64, ok bool) {
	if now, ok = readSysInt(base, "energy_now"); ok {
		full, _ = readSysInt(base, "energy_full")
		rate, _ = readSysInt(base, "power_now")

		return now, full, rate, true
	}

	if now, ok = readSysInt(base, "charge_now"); ok {
		full, _ = readSysInt(base, "charge_full")
		rate, _ = readSysInt(base, "current_now")

		return now, full, rate, true
	}

	return 0, 0, 0, false
}

func readSysInt(base, name string) (int64, bool) {
	value := readSysString(base, name)

	if value == "" {
		return 0, false
	}

	number, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		return 0, false
	}

	return number, true
}

func readSysString(base, name string) string {
	data, err := os.ReadFile(filepath.Join(base, name))

	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}
