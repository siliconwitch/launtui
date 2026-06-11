package widgets

import (
	"net/url"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WebConfig struct {
	Enabled   bool   `toml:"enabled"`
	SearchURL string `toml:"search_url"`
}

func (WebConfig) SectionName() string { return "web" }

func DefaultWebConfig() WebConfig {
	return WebConfig{Enabled: true, SearchURL: "https://duckduckgo.com/?q=%s"}
}

var (
	webSelectedStyle = lipgloss.NewStyle().Foreground(webColor).Bold(true)
	webBarStyle      = lipgloss.NewStyle().Foreground(webColor)
)

type webAction struct {
	label string
	url   string
}

type Web struct {
	cfg     WebConfig
	query   string
	actions []webAction
	cursor  int
}

func NewWeb(cfg WebConfig) Web {
	return Web{cfg: cfg}
}

func (Web) Name() string    { return "Web" }
func (Web) Hotkey() string  { return "ctrl+s" }
func (w Web) Enabled() bool { return w.cfg.Enabled }
func (Web) Init() tea.Cmd   { return nil }

func (w Web) Update(tea.Msg) (Mode, tea.Cmd) { return w, nil }

func (w Web) SetQuery(query string) Mode {
	w.query = strings.TrimSpace(query)
	w.cursor = 0
	w.actions = nil

	if w.query == "" {
		return w
	}

	if address, ok := queryAsURL(w.query); ok {
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

func queryAsURL(query string) (string, bool) {
	if strings.ContainsAny(query, " \t") {
		return "", false
	}

	if strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://") {
		return query, true
	}

	host, _, _ := strings.Cut(query, "/")

	if strings.HasPrefix(host, "localhost") {
		rest := strings.TrimPrefix(host, "localhost")

		if rest == "" || webPortPattern.MatchString(rest) {
			return "http://" + query, true
		}
	}

	if webHostPattern.MatchString(host) {
		return "https://" + query, true
	}

	return "", false
}

func (w Web) HasResults() bool {
	return len(w.actions) > 0
}

func (w Web) StrongMatch() bool {
	return len(w.actions) > 1
}

func (w Web) MoveUp() Mode {
	if w.cursor > 0 {
		w.cursor--
	}

	return w
}

func (w Web) MoveDown() Mode {
	if w.cursor < len(w.actions)-1 {
		w.cursor++
	}

	return w
}

func (w Web) Activate() tea.Cmd {
	if len(w.actions) == 0 {
		return nil
	}

	address := w.actions[w.cursor].url

	return func() tea.Msg {
		spawnDetached("xdg-open", address)

		return RequestQuitMsg{}
	}
}

func (w Web) View(width, rows int) string {
	if len(w.actions) == 0 {
		return subtleStyle.Render("type a web address or search query")
	}

	start, end := visibleRange(w.cursor, rows, len(w.actions))

	var lines []string

	for i := start; i < end; i++ {
		lines = append(lines, w.renderAction(w.actions[i], i == w.cursor, width))
	}

	return strings.Join(lines, "\n")
}

func (w Web) renderAction(action webAction, selected bool, width int) string {
	label := truncate(action.label, max(width-2, 1))

	if selected {
		return webBarStyle.Render("▌ ") + webSelectedStyle.Render(label)
	}

	return "  " + nameStyle.Render(label)
}
