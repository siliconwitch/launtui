package widgets

import (
	"math"
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
