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

type LauncherConfig struct {
	Placeholder string `toml:"placeholder"`
}

func (LauncherConfig) SectionName() string { return "launcher" }

func DefaultLauncherConfig() LauncherConfig {
	return LauncherConfig{Placeholder: "Search…"}
}

type desktopApp struct {
	Name    string
	Comment string
	Exec    string
}

type appsLoadedMsg []desktopApp

type Launcher struct {
	cfg      LauncherConfig
	input    textinput.Model
	apps     []desktopApp
	filtered []desktopApp
	cursor   int
	loaded   bool
}

func NewLauncher(cfg LauncherConfig) Launcher {
	input := textinput.New()
	input.Prompt = "❯ "
	input.Placeholder = cfg.Placeholder
	input.Focus()

	return Launcher{cfg: cfg, input: input}
}

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

		for i, app := range l.apps {
			names[i] = app.Name
		}

		matches := fuzzy.Find(query, names)
		l.filtered = make([]desktopApp, len(matches))

		for i, match := range matches {
			l.filtered[i] = l.apps[match.Index]
		}
	}

	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

func (l Launcher) InputView(width int) string {
	input := l.input

	if w := width - runeLen(input.Prompt) - 1; w >= 1 {
		input.Width = w
	}

	return input.View()
}

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
	avail := width - 2

	if avail < 1 {
		avail = 1
	}

	name, comment := app.Name, app.Comment

	if runeLen(name) > avail {
		name = truncate(name, avail)
		comment = ""
	}

	sub := ""

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

func loadAppsCmd() tea.Cmd {
	return func() tea.Msg {
		return appsLoadedMsg(scanDesktopApps())
	}
}

func launchCmd(app desktopApp) tea.Cmd {
	return func() tea.Msg {
		spawnDetached(app.Exec)

		return tea.QuitMsg{}
	}
}

func spawnDetached(cmdline string) {
	cmdline = strings.TrimSpace(cmdline)

	if cmdline == "" {
		return
	}

	cmd := exec.Command("sh", "-c", cmdline)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	_ = cmd.Start()
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
		app       desktopApp
		entryType string
		inEntry   bool
		noDisplay bool
		isHidden  bool
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
		case "NoDisplay":
			noDisplay = strings.TrimSpace(value) == "true"
		case "Hidden":
			isHidden = strings.TrimSpace(value) == "true"
		}
	}

	if scanner.Err() != nil {
		return desktopApp{}, false
	}

	if entryType != "Application" || noDisplay || isHidden || app.Name == "" || app.Exec == "" {
		return desktopApp{}, false
	}

	return app, true
}

func stripFieldCodes(execLine string) string {
	execLine = strings.ReplaceAll(execLine, "%%", "\x00")
	execLine = execFieldCodes.ReplaceAllString(execLine, "")
	execLine = strings.ReplaceAll(execLine, "\x00", "%")

	return strings.TrimSpace(execLine)
}
