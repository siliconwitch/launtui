package widgets

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ClockConfig struct {
	Enabled bool   `toml:"enabled"`
	Format  string `toml:"format"`
}

func (ClockConfig) SectionName() string { return "clock" }

func DefaultClockConfig() ClockConfig {
	return ClockConfig{Enabled: true, Format: "Mon 2 Jan - 15:04"}
}

var clockStyle = lipgloss.NewStyle().Foreground(clockColor).Bold(true)

type clockTickMsg time.Time

type Clock struct {
	cfg ClockConfig
	now time.Time
}

func NewClock(cfg ClockConfig) Clock {
	return Clock{cfg: cfg, now: time.Now()}
}

func (c Clock) Enabled() bool { return c.cfg.Enabled }

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

	return clockStyle.Render(c.now.Format(c.cfg.Format))
}

func clockTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return clockTickMsg(t)
	})
}
