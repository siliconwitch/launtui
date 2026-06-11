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
	batteryLevelStyle = lipgloss.NewStyle().Foreground(batteryColor).Bold(true)
	batteryInfoStyle  = lipgloss.NewStyle().Foreground(batteryColor)
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

	return readBatteryCmd(b.cfg.Device)
}

func (b Battery) Update(msg tea.Msg) (Battery, tea.Cmd) {
	reading, ok := msg.(batteryMsg)

	if !ok {
		return b, nil
	}

	b.reading = batteryReading(reading)

	return b, scheduleBatteryCmd(b.cfg.Device)
}

func (b Battery) View() string {
	if !b.cfg.Enabled || !b.reading.present {
		return ""
	}

	reading := b.reading

	line := batteryLevelStyle.Render(strconv.Itoa(reading.percent)+"%") +
		" " + batteryInfoStyle.Render(batteryIcon(reading))

	if reading.hours > 0 {
		line += " " + batteryInfoStyle.Render(batteryDuration(reading.hours))
	}

	return line
}

func batteryIcon(reading batteryReading) string {
	switch reading.status {
	case "Charging":
		return ""
	case "Not charging":
		return ""
	case "Full":
		return ""
	}

	return batteryLevelIcon(reading.percent)
}

func batteryLevelIcon(percent int) string {
	switch {
	case percent >= 90:
		return ""
	case percent >= 65:
		return ""
	case percent >= 40:
		return ""
	case percent >= 15:
		return ""
	default:
		return ""
	}
}

func batteryDuration(hours float64) string {
	if hours > 1.5 {
		return strings.TrimSuffix(strconv.FormatFloat(hours, 'f', 1, 64), ".0") + "h"
	}

	return strconv.Itoa(int(math.Round(hours*60))) + "m"
}

func readBatteryCmd(device string) tea.Cmd {
	return func() tea.Msg {
		return batteryMsg(readBattery(device))
	}
}

func scheduleBatteryCmd(device string) tea.Cmd {
	return tea.Tick(batteryInterval, func(time.Time) tea.Msg {
		return batteryMsg(readBattery(device))
	})
}

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
