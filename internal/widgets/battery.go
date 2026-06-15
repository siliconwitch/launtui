package widgets

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BatteryConfig struct {
	Enabled bool   `toml:"enabled"`
	Device  string `toml:"device"`
}

func (BatteryConfig) SectionName() string { return "battery" }

func DefaultBatteryConfig() BatteryConfig {
	return BatteryConfig{Enabled: true, Device: "BAT0"}
}

const batteryInterval = 10 * time.Second

var (
	batteryLevelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	batteryInfoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

type batteryReading struct {
	present bool
	status  string
	percent int
	hours   float64
}

type batteryMsg batteryReading

type Battery struct {
	cfg     BatteryConfig
	reading batteryReading
}

func NewBattery(cfg BatteryConfig) Battery {
	return Battery{cfg: cfg}
}

func (b Battery) Enabled() bool { return b.cfg.Enabled }

func (b Battery) Init() tea.Cmd {
	if !b.cfg.Enabled {
		return nil
	}

	return func() tea.Msg {
		return batteryMsg(readBattery(b.cfg.Device))
	}
}

func (b Battery) Update(msg tea.Msg) (Battery, tea.Cmd) {
	reading, ok := msg.(batteryMsg)

	if !ok {
		return b, nil
	}

	b.reading = batteryReading(reading)

	return b, tea.Tick(batteryInterval, func(time.Time) tea.Msg {
		return batteryMsg(readBattery(b.cfg.Device))
	})
}

func (b Battery) View() string {
	if !b.cfg.Enabled || !b.reading.present {
		return ""
	}

	reading := b.reading

	var icon string

	switch reading.status {
	case "Charging":
		icon = ""
	case "Not charging":
		icon = ""
	case "Full":
		icon = ""
	default:
		switch {
		case reading.percent >= 90:
			icon = ""
		case reading.percent >= 65:
			icon = ""
		case reading.percent >= 40:
			icon = ""
		case reading.percent >= 15:
			icon = ""
		default:
			icon = ""
		}
	}

	line := batteryLevelStyle.Render(strconv.Itoa(reading.percent)+"%") +
		" " + batteryInfoStyle.Render(icon)

	if reading.hours > 0 {
		duration := strconv.Itoa(int(math.Round(reading.hours*60))) + "m"

		if reading.hours > 1.5 {
			duration = strings.TrimSuffix(strconv.FormatFloat(reading.hours, 'f', 1, 64), ".0") + "h"
		}

		line += " " + batteryInfoStyle.Render(duration)
	}

	return line
}

func readBattery(device string) batteryReading {
	reading := readBatteryAt(filepath.Join("/sys/class/power_supply", device))

	if reading.present {
		return reading
	}

	entries, err := os.ReadDir("/sys/class/power_supply")

	if err != nil {
		return reading
	}

	for _, entry := range entries {
		base := filepath.Join("/sys/class/power_supply", entry.Name())

		if readSysString(base, "type") == "Battery" {
			return readBatteryAt(base)
		}
	}

	return reading
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

	now, ok := readSysInt(base, "energy_now")

	var full, rate int64

	if ok {
		full, _ = readSysInt(base, "energy_full")
		rate, _ = readSysInt(base, "power_now")
	} else if now, ok = readSysInt(base, "charge_now"); ok {
		full, _ = readSysInt(base, "charge_full")
		rate, _ = readSysInt(base, "current_now")
	}

	hours := 0.0

	if ok && rate > 0 {
		switch status {
		case "Discharging":
			hours = float64(now) / float64(rate)
		case "Charging":
			if full > now {
				hours = float64(full-now) / float64(rate)
			}
		}
	}

	return batteryReading{
		present: true,
		status:  status,
		percent: int(capacity),
		hours:   hours,
	}
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
