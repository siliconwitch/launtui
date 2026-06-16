package widgets

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/enescakir/emoji"
)

type EmojiConfig struct {
	Enabled bool `toml:"enabled"`
}

func (EmojiConfig) SectionName() string { return "emoji" }

func DefaultEmojiConfig() EmojiConfig {
	return EmojiConfig{Enabled: true}
}

var emojiAccent = lipgloss.Color("3")

type emojiEntry struct {
	glyph string
	name  string
}

type Emoji struct {
	cfg  EmojiConfig
	list list[emojiEntry]
}

func NewEmoji(cfg EmojiConfig) Emoji {
	catalogue := emoji.Map()
	entries := make([]emojiEntry, 0, len(catalogue))

	for code, glyph := range catalogue {
		name := strings.ReplaceAll(strings.Trim(code, ":"), "_", " ")
		entries = append(entries, emojiEntry{glyph: glyph, name: name})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	mode := Emoji{cfg: cfg, list: newList(func(entry emojiEntry) string { return entry.name })}
	mode.list.setItems(entries)

	return mode
}

func (Emoji) Name() string    { return "Emoji" }
func (Emoji) Hotkey() string  { return "ctrl+e" }
func (e Emoji) Enabled() bool { return e.cfg.Enabled }

func (Emoji) Init() tea.Cmd { return nil }

func (e Emoji) Update(tea.Msg) (Mode, tea.Cmd) { return e, nil }

func (e Emoji) SetQuery(query string) Mode {
	e.list.setQuery(query)

	return e
}

func (Emoji) Accent() lipgloss.Color { return emojiAccent }

func (e Emoji) Status() string {
	if len(e.list.filtered) == 0 {
		return subtleStyle.Render("no matching emoji")
	}

	return ""
}

func (e Emoji) Rows() []Row {
	return e.list.rows(func(entry emojiEntry) Row {
		return Row{left: entry.glyph + "  " + entry.name}
	})
}

func (e Emoji) Activate(index int) tea.Cmd {
	entry, ok := e.list.at(index)

	if !ok {
		return nil
	}

	return func() tea.Msg {
		copyToClipboard(entry.glyph)
		recordClipboardText(entry.glyph, 0)

		return RequestQuitMsg{}
	}
}
