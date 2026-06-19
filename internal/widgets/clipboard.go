package widgets

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ClipboardConfig struct {
	Enabled    bool `toml:"enabled"`
	MaxHistory int  `toml:"max_history"`
}

func (ClipboardConfig) SectionName() string { return "clipboard" }

func DefaultClipboardConfig() ClipboardConfig {
	return ClipboardConfig{Enabled: true, MaxHistory: 50}
}

var clipboardAccent = lipgloss.Color("6")

type clipboardHistoryMsg []clipboardEntry

type Clipboard struct {
	config ClipboardConfig
	list   list[clipboardEntry]
}

func NewClipboard(config ClipboardConfig) Clipboard {
	return Clipboard{config: config, list: newList(func(entry clipboardEntry) string { return clipboardPreview(entry.Text) })}
}

func (Clipboard) Name() string    { return "Clip" }
func (Clipboard) Hotkey() string  { return "ctrl+v" }
func (c Clipboard) Enabled() bool { return c.config.Enabled }

func (c Clipboard) Init() tea.Cmd {
	if !c.config.Enabled {
		return nil
	}

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
		cleaned := strings.TrimSpace(strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return ' '
			}

			return r
		}, line))

		if cleaned != "" {
			return cleaned
		}
	}

	return ""
}

func (Clipboard) Accent() lipgloss.Color { return clipboardAccent }

func (c Clipboard) Status() string {
	switch {
	case !c.list.loaded:
		return subtleStyle.Render("loading clipboard history…")
	case len(c.list.items) == 0:
		return subtleStyle.Render("clipboard history is empty — run `launtui -watch` to record copies")
	case len(c.list.filtered) == 0:
		return subtleStyle.Render("no matching clipboard entries")
	}

	return ""
}

func (c Clipboard) Rows() []Row {
	now := time.Now().Unix()

	return c.list.rows(func(entry clipboardEntry) Row {
		preview := clipboardPreview(entry.Text)

		if strings.Contains(strings.TrimSpace(entry.Text), "\n") {
			preview += " ⏎"
		}

		return Row{left: preview, right: subtleStyle.Render(relativeAge(now - entry.Time)), Deletable: true}
	})
}

func (c Clipboard) DeleteRow(index int) (Mode, tea.Cmd) {
	selected, ok := c.list.at(index)

	if !ok {
		return c, nil
	}

	entries := make([]clipboardEntry, 0, len(c.list.items))

	for _, entry := range c.list.items {
		if entry != selected {
			entries = append(entries, entry)
		}
	}

	c.list.setItems(entries)

	return c, saveClipboardHistoryCmd(entries)
}

func (c Clipboard) ClearRows() (Mode, tea.Cmd) {
	c.list.setItems(nil)

	return c, saveClipboardHistoryCmd(nil)
}

func saveClipboardHistoryCmd(entries []clipboardEntry) tea.Cmd {
	return func() tea.Msg {
		saveClipboardHistory(entries)

		return nil
	}
}

func (c Clipboard) Activate(index int) tea.Cmd {
	entry, ok := c.list.at(index)

	if !ok {
		return nil
	}

	limit := c.config.MaxHistory

	return func() tea.Msg {
		copyToClipboard(entry.Text)
		recordClipboardText(entry.Text, limit)

		return RequestQuitMsg{}
	}
}

func WatchClipboard(config ClipboardConfig) error {
	if !config.Enabled {
		return errors.New("clipboard mode is disabled in config")
	}

	self, err := os.Executable()

	if err != nil {
		return err
	}

	wlPaste, err := exec.LookPath("wl-paste")

	if err == nil {
		command := exec.Command(wlPaste, "--type", "text", "--no-newline", "--watch", self, "-record")
		command.Stderr = os.Stderr

		return command.Run()
	}

	last := ""

	for {
		text := readClipboard()

		if strings.TrimSpace(text) != "" && text != last {
			last = text
			recordClipboardText(text, config.MaxHistory)
		}

		time.Sleep(time.Second)
	}
}

func RecordClipboardStdin(config ClipboardConfig) error {
	if !config.Enabled {
		return nil
	}

	if wlPaste, err := exec.LookPath("wl-paste"); err == nil {
		if output, err := exec.Command(wlPaste, "--list-types").Output(); err == nil {
			if strings.Contains(string(output), "x-kde-passwordManagerHint") {
				return nil
			}
		}
	}

	data, err := io.ReadAll(io.LimitReader(os.Stdin, 256*1024))

	if err != nil {
		return err
	}

	recordClipboardText(string(data), config.MaxHistory)

	return nil
}
