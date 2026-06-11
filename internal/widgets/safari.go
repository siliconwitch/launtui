package widgets

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type SafariConfig struct {
	Enabled      bool `toml:"enabled"`
	HistoryLimit int  `toml:"history_limit"`
}

func (SafariConfig) SectionName() string { return "safari" }

func DefaultSafariConfig() SafariConfig {
	return SafariConfig{Enabled: true, HistoryLimit: 2000}
}

var (
	safariSelectedStyle = lipgloss.NewStyle().Foreground(safariColor).Bold(true)
	safariBarStyle      = lipgloss.NewStyle().Foreground(safariColor)
)

type safariKind int

const (
	safariTab safariKind = iota
	safariBookmark
	safariHistory
)

func (k safariKind) label() string {
	switch k {
	case safariTab:
		return "tab"
	case safariBookmark:
		return "bookmark"
	default:
		return "history"
	}
}

type safariEntry struct {
	kind   safariKind
	title  string
	url    string
	window int
	tab    int
}

type safariLoadedMsg struct {
	entries []safariEntry
	denied  bool
}

type Safari struct {
	cfg       SafariConfig
	available bool
	entries   []safariEntry
	filtered  []safariEntry
	query     string
	cursor    int
	loaded    bool
	denied    bool
}

func NewSafari(cfg SafariConfig) Safari {
	return Safari{cfg: cfg, available: safariSupported()}
}

func (Safari) Name() string    { return "Safari" }
func (Safari) Hotkey() string  { return "ctrl+b" }
func (s Safari) Enabled() bool { return s.cfg.Enabled && s.available }

func (s Safari) Init() tea.Cmd {
	if !s.Enabled() {
		return nil
	}

	return loadSafariCmd(s.cfg)
}

func loadSafariCmd(cfg SafariConfig) tea.Cmd {
	return func() tea.Msg {
		entries, denied := loadSafariEntries(cfg)

		return safariLoadedMsg{entries: entries, denied: denied}
	}
}

func (s Safari) Update(msg tea.Msg) (Mode, tea.Cmd) {
	loaded, ok := msg.(safariLoadedMsg)

	if !ok {
		return s, nil
	}

	s.entries = loaded.entries
	s.denied = loaded.denied
	s.loaded = true
	s.refilter()

	return s, nil
}

func (s Safari) SetQuery(query string) Mode {
	s.query = query
	s.cursor = 0
	s.refilter()

	return s
}

func (s *Safari) refilter() {
	query := strings.TrimSpace(s.query)

	if query == "" {
		s.filtered = s.entries
	} else {
		targets := make([]string, len(s.entries))

		for i, entry := range s.entries {
			targets[i] = entry.title + " " + entry.url
		}

		matches := fuzzy.Find(query, targets)
		s.filtered = make([]safariEntry, len(matches))

		for i, match := range matches {
			s.filtered[i] = s.entries[match.Index]
		}
	}

	if s.cursor >= len(s.filtered) {
		s.cursor = max(0, len(s.filtered)-1)
	}
}

func (s Safari) HasResults() bool {
	return s.loaded && len(s.filtered) > 0
}

func (s Safari) MoveUp() Mode {
	if s.cursor > 0 {
		s.cursor--
	}

	return s
}

func (s Safari) MoveDown() Mode {
	if s.cursor < len(s.filtered)-1 {
		s.cursor++
	}

	return s
}

func (s Safari) Activate() tea.Cmd {
	if len(s.filtered) == 0 {
		return nil
	}

	entry := s.filtered[s.cursor]

	return func() tea.Msg {
		activateSafariEntry(entry)

		return RequestQuitMsg{}
	}
}

func (s Safari) View(width, rows int) string {
	switch {
	case !s.loaded:
		return subtleStyle.Render("reading Safari…")
	case len(s.entries) == 0:
		if s.denied {
			return subtleStyle.Render("grant Full Disk Access (bookmarks, history) and Automation (tabs) to your terminal in System Settings > Privacy & Security")
		}

		return subtleStyle.Render("no Safari tabs, bookmarks, or history")
	case len(s.filtered) == 0:
		return subtleStyle.Render("no matching pages")
	}

	start, end := visibleRange(s.cursor, rows, len(s.filtered))

	var lines []string

	for i := start; i < end; i++ {
		lines = append(lines, s.renderEntry(s.filtered[i], i == s.cursor, width))
	}

	return strings.Join(lines, "\n")
}

func (s Safari) renderEntry(entry safariEntry, selected bool, width int) string {
	avail := max(width-2, 1)

	title := entry.title

	if title == "" {
		title = entry.url
	}

	kind := entry.kind.label()

	if displayWidth(title) > avail {
		title = truncate(title, avail)
		kind = ""
	}

	sub := ""

	if kind != "" {
		if gap := avail - displayWidth(title); gap > displayWidth(kind)+1 {
			sub = strings.Repeat(" ", gap-displayWidth(kind)) + subtleStyle.Render(kind)
		}
	}

	if selected {
		return safariBarStyle.Render("▌ ") + safariSelectedStyle.Render(title) + sub
	}

	return "  " + nameStyle.Render(title) + sub
}
