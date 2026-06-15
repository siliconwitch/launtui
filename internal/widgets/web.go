package widgets

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WebConfig struct {
	Enabled    bool   `toml:"enabled"`
	SearchURL  string `toml:"search_url"`
	MaxHistory int    `toml:"max_history"`
}

func (WebConfig) SectionName() string { return "web" }

func DefaultWebConfig() WebConfig {
	return WebConfig{Enabled: true, SearchURL: "https://duckduckgo.com/?q=%s", MaxHistory: 50}
}

const webHistoryFile = "web-history.json"

var webAccent = lipgloss.Color("12")

type webAction struct {
	label string
	url   string
}

type webVisit struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Time  int64  `json:"time"`
}

type webHistoryMsg []webVisit

type Web struct {
	cfg     WebConfig
	query   string
	actions []webAction
	history []webVisit
	cursor  int
}

func NewWeb(cfg WebConfig) Web {
	return Web{cfg: cfg}
}

func (Web) Name() string    { return "Web" }
func (Web) Hotkey() string  { return "ctrl+s" }
func (w Web) Enabled() bool { return w.cfg.Enabled }

func (w Web) Init() tea.Cmd {
	if !w.cfg.Enabled {
		return nil
	}

	return func() tea.Msg {
		path, err := launtuiDataPath(webHistoryFile)

		if err != nil {
			return webHistoryMsg(nil)
		}

		history, _ := loadJSON[[]webVisit](path)

		return webHistoryMsg(history)
	}
}

func (w Web) Update(msg tea.Msg) (Mode, tea.Cmd) {
	history, ok := msg.(webHistoryMsg)

	if !ok {
		return w, nil
	}

	w.history = history

	return w, nil
}

func (w Web) SetQuery(query string) Mode {
	w.query = strings.TrimSpace(query)
	w.cursor = 0
	w.actions = nil

	if w.query == "" {
		return w
	}

	address, isURL := "", false

	if !strings.ContainsAny(w.query, " \t") {
		switch {
		case strings.HasPrefix(w.query, "http://"), strings.HasPrefix(w.query, "https://"):
			address, isURL = w.query, true
		default:
			host, _, _ := strings.Cut(w.query, "/")

			if strings.HasPrefix(host, "localhost") {
				rest := strings.TrimPrefix(host, "localhost")

				if rest == "" || webPortPattern.MatchString(rest) {
					address, isURL = "http://"+w.query, true
				}
			}

			if !isURL && webHostPattern.MatchString(host) {
				address, isURL = "https://"+w.query, true
			}
		}
	}

	if isURL {
		w.actions = append(w.actions, webAction{label: "Open " + address, url: address})
	}

	search := strings.ReplaceAll(w.cfg.SearchURL, "%s", url.QueryEscape(w.query))
	w.actions = append(w.actions, webAction{label: "Search the web for “" + w.query + "”", url: search})

	return w
}

var (
	webHostPattern = regexp.MustCompile(`^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(:\d+)?$`)
	webPortPattern = regexp.MustCompile(`^:\d+$`)
)

func (w Web) HasResults() bool {
	return len(w.actions) > 0
}

func (w Web) StrongMatch() bool {
	return len(w.actions) > 1
}

func (w Web) itemCount() int {
	return len(w.actions) + len(w.history)
}

func (w Web) MoveUp() Mode {
	if w.cursor > 0 {
		w.cursor--
	}

	return w
}

func (w Web) MoveDown() Mode {
	if w.cursor < w.itemCount()-1 {
		w.cursor++
	}

	return w
}

func (w Web) Activate() tea.Cmd {
	var visit webVisit

	if w.cursor < len(w.actions) {
		action := w.actions[w.cursor]
		visit = webVisit{Label: action.label, URL: action.url, Time: time.Now().Unix()}
	} else if index := w.cursor - len(w.actions); index < len(w.history) {
		visit = w.history[index]
		visit.Time = time.Now().Unix()
	} else {
		return nil
	}

	limit := w.cfg.MaxHistory

	return func() tea.Msg {
		spawnDetached("", "xdg-open", visit.URL)

		if limit <= 0 {
			limit = 50
		}

		path, err := launtuiDataPath(webHistoryFile)

		if err == nil {
			previous, _ := loadJSON[[]webVisit](path)

			entries := prependCapped(previous, visit, limit, func(existing webVisit) bool {
				return existing.URL == visit.URL
			})

			_ = saveJSON(path, entries)
		}

		return RequestQuitMsg{}
	}
}

func (w Web) DeleteSelectedHistory() (Mode, tea.Cmd, bool) {
	index := w.cursor - len(w.actions)

	if index < 0 || index >= len(w.history) {
		return w, nil, false
	}

	w.history = removeAt(w.history, index)

	if w.cursor >= w.itemCount() {
		w.cursor = max(w.itemCount()-1, 0)
	}

	return w, saveWebHistoryCmd(w.history), true
}

func (w Web) ClearHistory() (Mode, tea.Cmd) {
	w.history = nil
	w.cursor = min(w.cursor, max(w.itemCount()-1, 0))

	return w, saveWebHistoryCmd(nil)
}

func saveWebHistoryCmd(history []webVisit) tea.Cmd {
	return func() tea.Msg {
		path, err := launtuiDataPath(webHistoryFile)

		if err != nil {
			return nil
		}

		_ = saveJSON(path, history)

		return nil
	}
}

func (w Web) View(width, rows int) string {
	if w.itemCount() == 0 {
		return subtleStyle.Render("type a web address or search query")
	}

	var lines []string

	for i, action := range w.actions {
		lines = append(lines, renderRow(webAccent, i == w.cursor, truncate(action.label, max(width-2, 1)), ""))
	}

	historyRows := rows - len(lines)

	if len(w.history) > 0 && historyRows > 0 {
		selected := w.cursor - len(w.actions)
		start, end := visibleRange(max(selected, 0), historyRows, len(w.history))

		for i := start; i < end; i++ {
			lines = append(lines, renderHistoryRow(webAccent, i == selected, truncate(w.history[i].Label, max(width-2, 1))))
		}
	}

	return strings.Join(lines, "\n")
}
