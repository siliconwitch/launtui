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
	Query string `json:"query"`
	Time  int64  `json:"time"`
}

type webHistoryMsg []webVisit

type Web struct {
	cfg     WebConfig
	query   string
	actions []webAction
	history []webVisit
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

func (Web) Accent() lipgloss.Color { return webAccent }

func (w Web) StrongMatch() bool {
	return len(w.actions) > 1
}

func (w Web) Status() string {
	if len(w.actions) == 0 && len(w.history) == 0 {
		return subtleStyle.Render("type a web address or search query")
	}

	return ""
}

func (w Web) Rows() []Row {
	rows := make([]Row, 0, len(w.actions)+len(w.history))

	for _, action := range w.actions {
		rows = append(rows, Row{left: action.label})
	}

	now := time.Now().Unix()

	for _, visit := range w.history {
		rows = append(rows, Row{
			left:      visit.Label,
			right:     subtleStyle.Render(relativeAge(now - visit.Time)),
			style:     rowDim,
			Deletable: true,
		})
	}

	return rows
}

func (w Web) Activate(index int) tea.Cmd {
	var visit webVisit

	if index < len(w.actions) {
		action := w.actions[index]
		visit = webVisit{Label: action.label, URL: action.url, Query: w.query, Time: time.Now().Unix()}
	} else if history := index - len(w.actions); history >= 0 && history < len(w.history) {
		visit = w.history[history]
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

func (w Web) RecallText(index int) (string, bool) {
	entry := index - len(w.actions)

	if entry < 0 || entry >= len(w.history) {
		return "", false
	}

	visit := w.history[entry]

	if visit.Query != "" {
		return visit.Query, true
	}

	return visit.URL, true
}

func (w Web) DeleteRow(index int) (Mode, tea.Cmd) {
	history := index - len(w.actions)

	if history < 0 || history >= len(w.history) {
		return w, nil
	}

	w.history = removeAt(w.history, history)

	return w, saveWebHistoryCmd(w.history)
}

func (w Web) ClearRows() (Mode, tea.Cmd) {
	w.history = nil

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
