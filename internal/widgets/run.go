package widgets

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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

type desktopApp struct {
	Name       string
	Comment    string
	Exec       string
	Terminal   bool
	WorkingDir string
}

type appsLoadedMsg []desktopApp

type Run struct {
	cfg      RunConfig
	apps     []desktopApp
	filtered []desktopApp
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

func (r Run) visibleApps(apps []desktopApp) []desktopApp {
	if len(r.cfg.Exclude) == 0 {
		return apps
	}

	excluded := make(map[string]bool, len(r.cfg.Exclude))

	for _, name := range r.cfg.Exclude {
		excluded[strings.ToLower(strings.TrimSpace(name))] = true
	}

	var kept []desktopApp

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
		r.filtered = make([]desktopApp, len(matches))

		for i, match := range matches {
			r.filtered[i] = r.apps[match.Index]
		}
	}

	if r.cursor >= len(r.filtered) {
		r.cursor = max(0, len(r.filtered)-1)
	}
}

func (r Run) renderApp(app desktopApp, selected bool, width int) string {
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
		return appsLoadedMsg(scanDesktopApps())
	}
}

func launchCmd(app desktopApp, preferredTerminal string) tea.Cmd {
	return func() tea.Msg {
		spawnDetachedIn(app.WorkingDir, launchArgv(app, preferredTerminal)...)

		return RequestQuitMsg{}
	}
}

func launchArgv(app desktopApp, preferredTerminal string) []string {
	cmdline := strings.TrimSpace(app.Exec)

	if cmdline == "" {
		return nil
	}

	if app.Terminal {
		return terminalArgv(resolveTerminal(preferredTerminal), cmdline)
	}

	return []string{"sh", "-c", cmdline}
}

func resolveTerminal(preferred string) string {
	if preferred != "" {
		return preferred
	}

	if terminal := os.Getenv("TERMINAL"); terminal != "" {
		return terminal
	}

	candidates := []string{
		"foot", "alacritty", "kitty", "ghostty", "wezterm",
		"gnome-terminal", "konsole", "xfce4-terminal", "xterm",
	}

	for _, candidate := range candidates {
		_, err := exec.LookPath(candidate)

		if err == nil {
			return candidate
		}
	}

	return ""
}

func terminalArgv(terminal, cmdline string) []string {
	if terminal == "" {
		return []string{"sh", "-c", cmdline}
	}

	switch filepath.Base(terminal) {
	case "foot", "kitty":
		return []string{terminal, "sh", "-c", cmdline}
	case "wezterm":
		return []string{terminal, "start", "--", "sh", "-c", cmdline}
	case "gnome-terminal":
		return []string{terminal, "--", "sh", "-c", cmdline}
	case "xfce4-terminal":
		return []string{terminal, "-x", "sh", "-c", cmdline}
	default:
		return []string{terminal, "-e", "sh", "-c", cmdline}
	}
}

func applicationDirs() []string {
	var dirs []string

	dataHome := os.Getenv("XDG_DATA_HOME")

	if dataHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}

	if dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "applications"))
	}

	dataDirs := os.Getenv("XDG_DATA_DIRS")

	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}

	for _, dir := range filepath.SplitList(dataDirs) {
		if dir != "" {
			dirs = append(dirs, filepath.Join(dir, "applications"))
		}
	}

	return dirs
}

func scanDesktopApps() []desktopApp {
	seen := map[string]bool{}

	var apps []desktopApp

	for _, dir := range applicationDirs() {
		entries, err := os.ReadDir(dir)

		if err != nil {
			continue
		}

		for _, entry := range entries {
			id := entry.Name()

			if entry.IsDir() || !strings.HasSuffix(id, ".desktop") || seen[id] {
				continue
			}

			seen[id] = true

			if app, ok := parseDesktopFile(filepath.Join(dir, id)); ok {
				apps = append(apps, app)
			}
		}
	}

	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps
}

var execFieldCodes = regexp.MustCompile(`%[fFuUdDnNickvm]`)

func parseDesktopFile(path string) (desktopApp, bool) {
	file, err := os.Open(path)

	if err != nil {
		return desktopApp{}, false
	}

	defer file.Close()

	var (
		app        desktopApp
		entryType  string
		inEntry    bool
		noDisplay  bool
		isHidden   bool
		tryExec    string
		onlyShowIn string
		notShowIn  string
	)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}

		if !inEntry {
			continue
		}

		key, value, ok := strings.Cut(line, "=")

		if !ok {
			continue
		}

		switch strings.TrimSpace(key) {
		case "Type":
			entryType = strings.TrimSpace(value)
		case "Name":
			app.Name = strings.TrimSpace(value)
		case "Comment":
			app.Comment = strings.TrimSpace(value)
		case "Exec":
			app.Exec = stripFieldCodes(value)
		case "TryExec":
			tryExec = strings.TrimSpace(value)
		case "Path":
			app.WorkingDir = strings.TrimSpace(value)
		case "Terminal":
			app.Terminal = strings.TrimSpace(value) == "true"
		case "NoDisplay":
			noDisplay = strings.TrimSpace(value) == "true"
		case "Hidden":
			isHidden = strings.TrimSpace(value) == "true"
		case "OnlyShowIn":
			onlyShowIn = strings.TrimSpace(value)
		case "NotShowIn":
			notShowIn = strings.TrimSpace(value)
		}
	}

	if scanner.Err() != nil {
		return desktopApp{}, false
	}

	if entryType != "Application" || noDisplay || isHidden || app.Name == "" || app.Exec == "" {
		return desktopApp{}, false
	}

	if !desktopVisibleIn(onlyShowIn, notShowIn, os.Getenv("XDG_CURRENT_DESKTOP")) {
		return desktopApp{}, false
	}

	if tryExec != "" {
		_, err := exec.LookPath(tryExec)

		if err != nil {
			return desktopApp{}, false
		}
	}

	return app, true
}

func desktopVisibleIn(onlyShowIn, notShowIn, currentDesktop string) bool {
	desktops := map[string]bool{}

	for _, name := range strings.Split(currentDesktop, ":") {
		if name != "" {
			desktops[strings.ToLower(name)] = true
		}
	}

	if onlyShowIn != "" {
		for _, name := range strings.Split(onlyShowIn, ";") {
			if name != "" && desktops[strings.ToLower(name)] {
				return true
			}
		}

		return false
	}

	for _, name := range strings.Split(notShowIn, ";") {
		if name != "" && desktops[strings.ToLower(name)] {
			return false
		}
	}

	return true
}

func stripFieldCodes(execLine string) string {
	execLine = strings.ReplaceAll(execLine, "%%", "\x00")
	execLine = execFieldCodes.ReplaceAllString(execLine, "")
	execLine = strings.ReplaceAll(execLine, `""`, "")
	execLine = strings.ReplaceAll(execLine, `''`, "")
	execLine = strings.ReplaceAll(execLine, "\x00", "%")

	return strings.TrimSpace(execLine)
}
