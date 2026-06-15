package widgets

import (
	"strings"
	"time"
	_ "time/tzdata"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ClockConfig struct {
	Enabled bool     `toml:"enabled"`
	Format  string   `toml:"format"`
	Zones   []string `toml:"zones"`
}

func (ClockConfig) SectionName() string { return "clock" }

func DefaultClockConfig() ClockConfig {
	return ClockConfig{Enabled: true, Format: "Mon 2 Jan - 15:04"}
}

var clockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)

type clockTickMsg time.Time

type Clock struct {
	cfg       ClockConfig
	now       time.Time
	locations []*time.Location
	current   int
}

func NewClock(cfg ClockConfig) Clock {
	locations := []*time.Location{time.Local}

	for _, zone := range cfg.Zones {
		if location, err := time.LoadLocation(zone); err == nil {
			locations = append(locations, location)
		}
	}

	return Clock{cfg: cfg, now: time.Now(), locations: locations}
}

func (c Clock) Enabled() bool { return c.cfg.Enabled }

func (c Clock) NextZone() Clock {
	if len(c.locations) > 1 {
		c.current = (c.current + 1) % len(c.locations)
	}

	return c
}

func (c Clock) Init() tea.Cmd {
	if !c.cfg.Enabled {
		return nil
	}

	return clockTick()
}

func (c Clock) Update(msg tea.Msg) (Clock, tea.Cmd) {
	tick, ok := msg.(clockTickMsg)

	if !ok {
		return c, nil
	}

	c.now = time.Time(tick)

	return c, clockTick()
}

func (c Clock) View() string {
	if !c.cfg.Enabled {
		return ""
	}

	location := c.locations[c.current]
	text := c.now.In(location).Format(c.cfg.Format)

	if c.current != 0 {
		name := location.String()

		if index := strings.LastIndex(name, "/"); index >= 0 {
			name = name[index+1:]
		}

		text += " " + strings.ReplaceAll(name, "_", " ")
	}

	return clockStyle.Render(text)
}

func clockTick() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return clockTickMsg(t)
	})
}
