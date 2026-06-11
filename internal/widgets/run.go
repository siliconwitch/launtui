package widgets

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type RunConfig struct {
	Enabled  bool     `toml:"enabled"`
	Exclude  []string `toml:"exclude"`
	Terminal string   `toml:"terminal"`
}

func (RunConfig) SectionName() string { return "run" }

func DefaultRunConfig() RunConfig {
	return RunConfig{Enabled: true}
}

var (
	selNameStyle = lipgloss.NewStyle().Foreground(launcherColor).Bold(true)
	selBarStyle  = lipgloss.NewStyle().Foreground(launcherColor)
)

type runApp struct {
	Name       string
	Comment    string
	Exec       string
	Terminal   bool
	WorkingDir string
}

type appsLoadedMsg []runApp

type Run struct {
	cfg      RunConfig
	apps     []runApp
	filtered []runApp
	query    string
	cursor   int
	loaded   bool
}

func NewRun(cfg RunConfig) Run {
	return Run{cfg: cfg}
}

func (Run) Name() string    { return "Run" }
func (Run) Hotkey() string  { return "ctrl+r" }
func (r Run) Enabled() bool { return r.cfg.Enabled }

func (r Run) Init() tea.Cmd {
	if !r.cfg.Enabled {
		return nil
	}

	return loadAppsCmd()
}

func (r Run) Update(msg tea.Msg) (Mode, tea.Cmd) {
	loaded, ok := msg.(appsLoadedMsg)

	if !ok {
		return r, nil
	}

	r.apps = r.visibleApps(loaded)
	r.loaded = true
	r.refilter()

	return r, nil
}

func (r Run) visibleApps(apps []runApp) []runApp {
	if len(r.cfg.Exclude) == 0 {
		return apps
	}

	excluded := make(map[string]bool, len(r.cfg.Exclude))

	for _, name := range r.cfg.Exclude {
		excluded[strings.ToLower(strings.TrimSpace(name))] = true
	}

	var kept []runApp

	for _, app := range apps {
		if !excluded[strings.ToLower(strings.TrimSpace(app.Name))] {
			kept = append(kept, app)
		}
	}

	return kept
}

func (r Run) SetQuery(query string) Mode {
	r.query = query
	r.cursor = 0
	r.refilter()

	return r
}

func (r Run) HasResults() bool {
	return r.loaded && len(r.filtered) > 0
}

func (r Run) MoveUp() Mode {
	if r.cursor > 0 {
		r.cursor--
	}

	return r
}

func (r Run) MoveDown() Mode {
	if r.cursor < len(r.filtered)-1 {
		r.cursor++
	}

	return r
}

func (r Run) Activate() tea.Cmd {
	if len(r.filtered) == 0 {
		return nil
	}

	return launchCmd(r.filtered[r.cursor], r.cfg.Terminal)
}

func (r Run) View(width, rows int) string {
	switch {
	case !r.loaded:
		return subtleStyle.Render("scanning applications…")
	case len(r.filtered) == 0:
		return subtleStyle.Render("no matching applications")
	}

	start, end := visibleRange(r.cursor, rows, len(r.filtered))

	var b strings.Builder

	for i := start; i < end; i++ {
		if i > start {
			b.WriteByte('\n')
		}

		b.WriteString(r.renderApp(r.filtered[i], i == r.cursor, width))
	}

	return b.String()
}

func (r *Run) refilter() {
	query := strings.TrimSpace(r.query)

	if query == "" {
		r.filtered = r.apps
	} else {
		names := make([]string, len(r.apps))

		for i, app := range r.apps {
			names[i] = app.Name
		}

		matches := fuzzy.Find(query, names)
		r.filtered = make([]runApp, len(matches))

		for i, match := range matches {
			r.filtered[i] = r.apps[match.Index]
		}
	}

	if r.cursor >= len(r.filtered) {
		r.cursor = max(0, len(r.filtered)-1)
	}
}

func (r Run) renderApp(app runApp, selected bool, width int) string {
	avail := width - 2

	if avail < 1 {
		avail = 1
	}

	name, comment := app.Name, app.Comment

	if displayWidth(name) > avail {
		name = truncate(name, avail)
		comment = ""
	}

	sub := ""

	if comment != "" {
		if gap := avail - displayWidth(name); gap > 3 {
			comment = truncate(comment, gap-2)
			sub = strings.Repeat(" ", gap-displayWidth(comment)) + subtleStyle.Render(comment)
		}
	}

	if selected {
		return selBarStyle.Render("▌ ") + selNameStyle.Render(name) + sub
	}

	return "  " + nameStyle.Render(name) + sub
}

func loadAppsCmd() tea.Cmd {
	return func() tea.Msg {
		return appsLoadedMsg(scanInstalledApps())
	}
}

func launchCmd(app runApp, preferredTerminal string) tea.Cmd {
	return func() tea.Msg {
		spawnDetachedIn(app.WorkingDir, launchArgv(app, preferredTerminal)...)

		return RequestQuitMsg{}
	}
}
