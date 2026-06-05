package widgets

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ClockConfig configures the clock.
type ClockConfig struct {
	Format string `toml:"format"` // Go reference-time layout for the time line
}

func (ClockConfig) SectionName() string { return "clock" }

func DefaultClockConfig() ClockConfig {
	return ClockConfig{Format: "15:04:05"}
}

// clockTickMsg advances the clock; one is delivered every second.
type clockTickMsg time.Time

// Clock shows the current time over the date. It's purely decorative: it
// consumes only its own tick and ignores every other message.
type Clock struct {
	cfg ClockConfig
	now time.Time
}

func NewClock(cfg ClockConfig) Clock {
	return Clock{cfg: cfg, now: time.Now()}
}

// Init starts the one-second tick.
func (c Clock) Init() tea.Cmd { return clockTick() }

func (c Clock) Update(msg tea.Msg) (Clock, tea.Cmd) {
	if t, ok := msg.(clockTickMsg); ok {
		c.now = time.Time(t)
		return c, clockTick()
	}
	return c, nil
}

// View renders the time over the date as a right-aligned two-line block, ready
// to be pinned into a corner by the caller.
func (c Clock) View() string {
	return lipgloss.JoinVertical(lipgloss.Right,
		clockTimeStyle.Render(c.now.Format(c.cfg.Format)),
		clockDateStyle.Render(c.now.Format("Mon 2 Jan 2006")),
	)
}

func clockTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return clockTickMsg(t)
	})
}
