package widgets

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ClipboardConfig struct {
	Enabled  bool `toml:"enabled"`
	MaxItems int  `toml:"max_items"`
}

func (ClipboardConfig) SectionName() string { return "clipboard" }

func DefaultClipboardConfig() ClipboardConfig {
	return ClipboardConfig{Enabled: true, MaxItems: defaultClipboardLimit}
}

var clipboardAccent = lipgloss.Color("6")

type clipboardHistoryMsg []clipboardEntry

type Clipboard struct {
	cfg  ClipboardConfig
	list list[clipboardEntry]
}

func NewClipboard(cfg ClipboardConfig) Clipboard {
	return Clipboard{cfg: cfg, list: newList(func(entry clipboardEntry) string { return clipboardPreview(entry.Text) })}
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

	c.list.setItems(history)

	return c, nil
}

func (c Clipboard) SetQuery(query string) Mode {
	c.list.setQuery(query)

	return c
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

func (c Clipboard) HasResults() bool { return c.list.hasResults() }

func (c Clipboard) MoveUp() Mode {
	c.list.moveUp()

	return c
}

func (c Clipboard) MoveDown() Mode {
	c.list.moveDown()

	return c
}

func (c Clipboard) DeleteSelectedHistory() (Mode, tea.Cmd, bool) {
	selected, ok := c.list.selected()

	if !ok {
		return c, nil, false
	}

	entries := make([]clipboardEntry, 0, len(c.list.items))

	for _, entry := range c.list.items {
		if entry != selected {
			entries = append(entries, entry)
		}
	}

	c.list.setItems(entries)

	return c, saveClipboardHistoryCmd(entries), true
}

func (c Clipboard) ClearHistory() (Mode, tea.Cmd) {
	c.list.setItems(nil)

	return c, saveClipboardHistoryCmd(nil)
}

func saveClipboardHistoryCmd(entries []clipboardEntry) tea.Cmd {
	return func() tea.Msg {
		saveClipboardHistory(entries)

		return nil
	}
}

func (c Clipboard) Activate() tea.Cmd {
	entry, ok := c.list.selected()

	if !ok {
		return nil
	}

	limit := c.cfg.MaxItems

	return func() tea.Msg {
		copyToClipboard(entry.Text)
		recordClipboardText(entry.Text, limit)

		return RequestQuitMsg{}
	}
}

func (c Clipboard) View(width, rows int) string {
	switch {
	case !c.list.loaded:
		return subtleStyle.Render("loading clipboard history…")
	case len(c.list.items) == 0:
		return subtleStyle.Render("clipboard history is empty — run `launtui -watch` to record copies")
	case len(c.list.filtered) == 0:
		return subtleStyle.Render("no matching clipboard entries")
	}

	now := time.Now().Unix()

	return c.list.view(width, rows, func(entry clipboardEntry, selected bool, width int) string {
		return c.renderEntry(entry, selected, width, now)
	})
}

func (c Clipboard) renderEntry(entry clipboardEntry, selected bool, width int, now int64) string {
	avail := max(width-2, 1)

	age := timeAgo(entry.Time, now)
	preview := clipboardPreview(entry.Text)

	if lines := strings.Count(strings.TrimSpace(entry.Text), "\n"); lines > 0 {
		preview += " ⏎"
	}

	if lipgloss.Width(preview) > avail {
		preview = truncate(preview, avail)
		age = ""
	}

	sub := ""

	if age != "" {
		if gap := avail - lipgloss.Width(preview); gap > lipgloss.Width(age)+1 {
			sub = strings.Repeat(" ", gap-lipgloss.Width(age)) + subtleStyle.Render(age)
		}
	}

	return renderRow(clipboardAccent, selected, preview, sub)
}

func timeAgo(unix, now int64) string {
	elapsed := now - unix

	switch {
	case elapsed < 60:
		return "now"
	case elapsed < 3600:
		return strconv.FormatInt(elapsed/60, 10) + "m"
	case elapsed < 86400:
		return strconv.FormatInt(elapsed/3600, 10) + "h"
	default:
		return strconv.FormatInt(elapsed/86400, 10) + "d"
	}
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
