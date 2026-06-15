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
)

type RunConfig struct {
	Enabled  bool     `toml:"enabled"`
	Comment  bool     `toml:"comment"`
	Exclude  []string `toml:"exclude"`
	Terminal string   `toml:"terminal"`
}

func (RunConfig) SectionName() string { return "run" }

func DefaultRunConfig() RunConfig {
	return RunConfig{Enabled: true, Comment: true}
}

var runAccent = lipgloss.Color("4")

type desktopApp struct {
	Name       string
	Comment    string
	Exec       string
	Terminal   bool
	WorkingDir string
}

type appsLoadedMsg []desktopApp

type Run struct {
	cfg  RunConfig
	list list[desktopApp]
}

func NewRun(cfg RunConfig) Run {
	return Run{cfg: cfg, list: newList(func(app desktopApp) string { return app.Name })}
}

func (Run) Name() string    { return "Run" }
func (Run) Hotkey() string  { return "ctrl+r" }
func (r Run) Enabled() bool { return r.cfg.Enabled }

func (r Run) Init() tea.Cmd {
	if !r.cfg.Enabled {
		return nil
	}

	return func() tea.Msg {
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

		seen := map[string]bool{}

		var apps []desktopApp

		for _, dir := range dirs {
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

		return appsLoadedMsg(apps)
	}
}

func (r Run) Update(msg tea.Msg) (Mode, tea.Cmd) {
	loaded, ok := msg.(appsLoadedMsg)

	if !ok {
		return r, nil
	}

	if len(r.cfg.Exclude) == 0 {
		r.list.setItems(loaded)

		return r, nil
	}

	excluded := make(map[string]bool, len(r.cfg.Exclude))

	for _, name := range r.cfg.Exclude {
		excluded[strings.ToLower(strings.TrimSpace(name))] = true
	}

	var kept []desktopApp

	for _, app := range loaded {
		if !excluded[strings.ToLower(strings.TrimSpace(app.Name))] {
			kept = append(kept, app)
		}
	}

	r.list.setItems(kept)

	return r, nil
}

func (r Run) SetQuery(query string) Mode {
	r.list.setQuery(query)

	return r
}

func (Run) Accent() lipgloss.Color { return runAccent }

func (r Run) Status() string {
	switch {
	case !r.list.loaded:
		return subtleStyle.Render("scanning applications…")
	case len(r.list.filtered) == 0:
		return subtleStyle.Render("no matching applications")
	}

	return ""
}

func (r Run) Rows() []Row {
	return r.list.rows(func(app desktopApp) Row {
		right := ""

		if r.cfg.Comment && app.Comment != "" {
			right = subtleStyle.Render(app.Comment)
		}

		return Row{left: app.Name, right: right}
	})
}

func (r Run) Activate(index int) tea.Cmd {
	app, ok := r.list.at(index)

	if !ok {
		return nil
	}

	return func() tea.Msg {
		cmdline := strings.TrimSpace(app.Exec)

		argv := []string{"sh", "-c", cmdline}

		if cmdline == "" {
			argv = nil
		} else if app.Terminal {
			argv = terminalArgv(resolveTerminal(r.cfg.Terminal), cmdline)
		}

		spawnDetached(app.WorkingDir, argv...)

		return RequestQuitMsg{}
	}
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
