package widgets

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type ClipboardConfig struct {
	Enabled  bool `toml:"enabled"`
	MaxItems int  `toml:"max_items"`
}

func (ClipboardConfig) SectionName() string { return "clipboard" }

func DefaultClipboardConfig() ClipboardConfig {
	return ClipboardConfig{Enabled: true, MaxItems: defaultClipboardLimit}
}

var (
	clipboardSelectedStyle = lipgloss.NewStyle().Foreground(clipboardColor).Bold(true)
	clipboardBarStyle      = lipgloss.NewStyle().Foreground(clipboardColor)
)

type clipboardHistoryMsg []clipboardEntry

type Clipboard struct {
	cfg      ClipboardConfig
	entries  []clipboardEntry
	filtered []clipboardEntry
	query    string
	cursor   int
	loaded   bool
}

func NewClipboard(cfg ClipboardConfig) Clipboard {
	return Clipboard{cfg: cfg}
}

func (Clipboard) Name() string    { return "Clip" }
func (Clipboard) Hotkey() string  { return "ctrl+v" }
func (c Clipboard) Enabled() bool { return c.cfg.Enabled }

func (c Clipboard) Init() tea.Cmd {
	if !c.cfg.Enabled {
		return nil
	}

	return loadClipboardCmd()
}

func loadClipboardCmd() tea.Cmd {
	return func() tea.Msg {
		return clipboardHistoryMsg(loadClipboardHistory())
	}
}

func (c Clipboard) Update(msg tea.Msg) (Mode, tea.Cmd) {
	history, ok := msg.(clipboardHistoryMsg)

	if !ok {
		return c, nil
	}

	c.entries = history
	c.loaded = true
	c.refilter()

	return c, nil
}

func (c Clipboard) SetQuery(query string) Mode {
	c.query = query
	c.cursor = 0
	c.refilter()

	return c
}

func (c *Clipboard) refilter() {
	query := strings.TrimSpace(c.query)

	if query == "" {
		c.filtered = c.entries
	} else {
		previews := make([]string, len(c.entries))

		for i, entry := range c.entries {
			previews[i] = clipboardPreview(entry.Text)
		}

		matches := fuzzy.Find(query, previews)
		c.filtered = make([]clipboardEntry, len(matches))

		for i, match := range matches {
			c.filtered[i] = c.entries[match.Index]
		}
	}

	if c.cursor >= len(c.filtered) {
		c.cursor = max(0, len(c.filtered)-1)
	}
}

func clipboardPreview(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed != "" {
			return trimmed
		}
	}

	return strings.TrimSpace(text)
}

func (c Clipboard) HasResults() bool {
	return c.loaded && len(c.filtered) > 0
}

func (c Clipboard) MoveUp() Mode {
	if c.cursor > 0 {
		c.cursor--
	}

	return c
}

func (c Clipboard) MoveDown() Mode {
	if c.cursor < len(c.filtered)-1 {
		c.cursor++
	}

	return c
}

func (c Clipboard) Activate() tea.Cmd {
	if len(c.filtered) == 0 {
		return nil
	}

	text := c.filtered[c.cursor].Text
	limit := c.cfg.MaxItems

	return func() tea.Msg {
		copyToClipboard(text)
		recordClipboardText(text, limit)

		return RequestQuitMsg{}
	}
}

func (c Clipboard) View(width, rows int) string {
	switch {
	case !c.loaded:
		return subtleStyle.Render("loading clipboard history…")
	case len(c.entries) == 0:
		return subtleStyle.Render("clipboard history is empty — run `launtui -watch` to record copies")
	case len(c.filtered) == 0:
		return subtleStyle.Render("no matching clipboard entries")
	}

	start, end := visibleRange(c.cursor, rows, len(c.filtered))
	now := time.Now().Unix()

	var b strings.Builder

	for i := start; i < end; i++ {
		if i > start {
			b.WriteByte('\n')
		}

		b.WriteString(c.renderEntry(c.filtered[i], i == c.cursor, width, now))
	}

	return b.String()
}

func (c Clipboard) renderEntry(entry clipboardEntry, selected bool, width int, now int64) string {
	avail := max(width-2, 1)

	age := timeAgo(entry.Time, now)
	preview := clipboardPreview(entry.Text)

	if lines := strings.Count(strings.TrimSpace(entry.Text), "\n"); lines > 0 {
		preview += " ⏎"
	}

	if displayWidth(preview) > avail {
		preview = truncate(preview, avail)
		age = ""
	}

	sub := ""

	if age != "" {
		if gap := avail - displayWidth(preview); gap > displayWidth(age)+1 {
			sub = strings.Repeat(" ", gap-displayWidth(age)) + subtleStyle.Render(age)
		}
	}

	if selected {
		return clipboardBarStyle.Render("▌ ") + clipboardSelectedStyle.Render(preview) + sub
	}

	return "  " + nameStyle.Render(preview) + sub
}

func WatchClipboard(cfg ClipboardConfig) error {
	if !cfg.Enabled {
		return errors.New("clipboard mode is disabled in config")
	}

	self, err := os.Executable()

	if err != nil {
		return err
	}

	wlPaste, err := exec.LookPath("wl-paste")

	if err == nil {
		cmd := exec.Command(wlPaste, "--type", "text", "--no-newline", "--watch", self, "-record")
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	last := ""

	for {
		text := readClipboard()

		if strings.TrimSpace(text) != "" && text != last {
			last = text
			recordClipboardText(text, cfg.MaxItems)
		}

		time.Sleep(time.Second)
	}
}

func RecordClipboardStdin(cfg ClipboardConfig) error {
	if !cfg.Enabled || clipboardMarkedSensitive() {
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(os.Stdin, 256*1024))

	if err != nil {
		return err
	}

	recordClipboardText(string(data), cfg.MaxItems)

	return nil
}

func clipboardMarkedSensitive() bool {
	wlPaste, err := exec.LookPath("wl-paste")

	if err != nil {
		return false
	}

	output, err := exec.Command(wlPaste, "--list-types").Output()

	if err != nil {
		return false
	}

	return strings.Contains(string(output), "x-kde-passwordManagerHint")
}
