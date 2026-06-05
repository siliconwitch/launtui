// Package widgets holds the individual launtui features. Each widget lives in
// its own file and owns its config: a Config struct (with `toml` tags), a
// SectionName naming its [table] in config.toml, and a Default constructor.
package widgets

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// LauncherConfig configures the program launcher.
type LauncherConfig struct {
	Placeholder string `toml:"placeholder"`
}

func (LauncherConfig) SectionName() string { return "launcher" }

func DefaultLauncherConfig() LauncherConfig {
	return LauncherConfig{Placeholder: "Search…"}
}

// desktopApp is one launchable application parsed from a .desktop file.
type desktopApp struct {
	Name    string
	Comment string
	Exec    string // Exec line with field codes (%u, %f, …) removed
}

// appsLoadedMsg carries the scanned applications back to the Launcher.
type appsLoadedMsg []desktopApp

// Launcher lists installed applications (.desktop files), fuzzy-searches them by
// name, and launches the selection detached.
type Launcher struct {
	cfg      LauncherConfig
	input    textinput.Model
	apps     []desktopApp // every app, sorted by name
	filtered []desktopApp // current matches
	cursor   int
	loaded   bool
}

func NewLauncher(cfg LauncherConfig) Launcher {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.Placeholder = cfg.Placeholder
	ti.Focus()
	return Launcher{cfg: cfg, input: ti}
}

// Init focuses the input and kicks off the (async) application scan.
func (l Launcher) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, loadAppsCmd())
}

func (l Launcher) Update(msg tea.Msg) (Launcher, tea.Cmd) {
	switch msg := msg.(type) {
	case appsLoadedMsg:
		l.apps = []desktopApp(msg)
		l.loaded = true
		l.refilter()
		return l, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "ctrl+p":
			if l.cursor > 0 {
				l.cursor--
			}
			return l, nil
		case "down", "ctrl+n":
			if l.cursor < len(l.filtered)-1 {
				l.cursor++
			}
			return l, nil
		case "enter":
			if len(l.filtered) > 0 {
				return l, launchCmd(l.filtered[l.cursor])
			}
			return l, nil
		}
	}

	// Anything else (typing, blink, …) goes to the text input; refilter when
	// the query actually changed.
	prev := l.input.Value()
	var cmd tea.Cmd
	l.input, cmd = l.input.Update(msg)
	if l.input.Value() != prev {
		l.refilter()
	}
	return l, cmd
}

func (l *Launcher) refilter() {
	query := strings.TrimSpace(l.input.Value())
	if query == "" {
		l.filtered = l.apps
	} else {
		names := make([]string, len(l.apps))
		for i, a := range l.apps {
			names[i] = a.Name
		}
		matches := fuzzy.Find(query, names)
		l.filtered = make([]desktopApp, len(matches))
		for i, m := range matches {
			l.filtered[i] = l.apps[m.Index]
		}
	}
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

// InputView renders the search line constrained to the given width.
func (l Launcher) InputView(width int) string {
	in := l.input // copy so we can size it for this frame without mutating state
	if w := width - runeLen(in.Prompt) - 1; w >= 1 {
		in.Width = w
	}
	return in.View()
}

// ListView renders up to rows results within width, scrolled to keep the cursor
// visible.
func (l Launcher) ListView(width, rows int) string {
	switch {
	case !l.loaded:
		return subtleStyle.Render("scanning applications…")
	case len(l.filtered) == 0:
		return subtleStyle.Render("no matching applications")
	}
	if rows < 1 {
		rows = 1
	}

	start := 0
	if l.cursor >= rows {
		start = l.cursor - rows + 1
	}
	end := min(start+rows, len(l.filtered))

	var b strings.Builder
	for i := start; i < end; i++ {
		if i > start {
			b.WriteByte('\n')
		}
		b.WriteString(l.renderApp(l.filtered[i], i == l.cursor, width))
	}
	return b.String()
}

func (l Launcher) renderApp(app desktopApp, selected bool, width int) string {
	avail := width - 2 // 2-column selection prefix
	if avail < 1 {
		avail = 1
	}

	name, comment := app.Name, app.Comment
	if runeLen(name) > avail {
		name = truncate(name, avail)
		comment = ""
	}

	// Right-align the comment in whatever space the name leaves.
	var sub string
	if comment != "" {
		if gap := avail - runeLen(name); gap > 3 {
			comment = truncate(comment, gap-2)
			sub = strings.Repeat(" ", gap-runeLen(comment)) + subtleStyle.Render(comment)
		}
	}

	if selected {
		return selBarStyle.Render("▌ ") + selNameStyle.Render(name) + sub
	}
	return "  " + nameStyle.Render(name) + sub
}

// --- application discovery & launching ---------------------------------------

func loadAppsCmd() tea.Cmd {
	return func() tea.Msg { return appsLoadedMsg(scanDesktopApps()) }
}

func launchCmd(app desktopApp) tea.Cmd {
	return func() tea.Msg {
		spawnDetached(app.Exec)
		return tea.QuitMsg{} // app launched; tear down the TUI
	}
}

// spawnDetached runs a command line in its own session so it survives launtui
// exiting (and its terminal closing). sh re-parses the line, handling quoting.
func spawnDetached(cmdline string) {
	cmdline = strings.TrimSpace(cmdline)
	if cmdline == "" {
		return
	}
	cmd := exec.Command("sh", "-c", cmdline)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()
}

// applicationDirs returns the XDG application directories in precedence order
// (most specific first).
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
	for _, d := range filepath.SplitList(dataDirs) {
		if d != "" {
			dirs = append(dirs, filepath.Join(d, "applications"))
		}
	}
	return dirs
}

func scanDesktopApps() []desktopApp {
	seen := map[string]bool{} // by .desktop id; first (most specific) wins
	var apps []desktopApp

	for _, dir := range applicationDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			id := e.Name()
			if e.IsDir() || !strings.HasSuffix(id, ".desktop") || seen[id] {
				continue
			}
			seen[id] = true // shadow any same-id file in a later dir, per XDG
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

// parseDesktopFile reads the [Desktop Entry] group and returns a launchable app,
// or ok=false if it shouldn't be listed (wrong type, hidden, or incomplete).
func parseDesktopFile(path string) (desktopApp, bool) {
	f, err := os.Open(path)
	if err != nil {
		return desktopApp{}, false
	}
	defer f.Close()

	var (
		app                 desktopApp
		typ                 string
		inEntry             bool
		noDisplay, isHidden bool
	)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
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

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		// Only the unlocalised keys (no "Name[xx]" suffix) are taken.
		switch strings.TrimSpace(key) {
		case "Type":
			typ = strings.TrimSpace(val)
		case "Name":
			app.Name = strings.TrimSpace(val)
		case "Comment":
			app.Comment = strings.TrimSpace(val)
		case "Exec":
			app.Exec = stripFieldCodes(val)
		case "NoDisplay":
			noDisplay = strings.TrimSpace(val) == "true"
		case "Hidden":
			isHidden = strings.TrimSpace(val) == "true"
		}
	}
	if sc.Err() != nil {
		return desktopApp{}, false
	}

	if typ != "Application" || noDisplay || isHidden || app.Name == "" || app.Exec == "" {
		return desktopApp{}, false
	}
	return app, true
}

func stripFieldCodes(exec string) string {
	exec = strings.ReplaceAll(exec, "%%", "\x00") // protect literal percents first
	exec = execFieldCodes.ReplaceAllString(exec, "")
	exec = strings.ReplaceAll(exec, "\x00", "%")
	return strings.TrimSpace(exec)
}
