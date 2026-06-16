package widgets

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProjectsConfig struct {
	Enabled        bool     `toml:"enabled"`
	Dirs           []string `toml:"dirs"`
	Projects       []string `toml:"projects"`
	Editor         string   `toml:"editor"`
	Terminal       string   `toml:"terminal"`
	EditorTerminal *bool    `toml:"editor_terminal"`
}

func (ProjectsConfig) SectionName() string { return "projects" }

func DefaultProjectsConfig() ProjectsConfig {
	return ProjectsConfig{Enabled: true, Dirs: []string{"~/Documents"}}
}

const projectFetchTimeout = 10 * time.Second

var (
	projectsAccent   = lipgloss.Color("2")
	cleanBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	dirtyBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	aheadBehindStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	projectSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

type project struct {
	name        string
	label       string
	path        string
	git         bool
	statusKnown bool
	branch      string
	dirty       bool
	ahead       int
	behind      int
	fetchFailed bool
}

type gitStatus struct {
	branch string
	dirty  bool
	ahead  int
	behind int
}

type projectsLoadedMsg []project

type projectStatusMsg struct {
	path        string
	status      gitStatus
	fetchFailed bool
}

type projectsTickMsg struct{}

type editorMissingMsg struct{}

type Projects struct {
	cfg       ProjectsConfig
	list      list[project]
	pending   int
	frame     int
	errorText string
}

func NewProjects(cfg ProjectsConfig) Projects {
	return Projects{cfg: cfg, list: newList(func(item project) string { return item.name })}
}

func (Projects) Name() string    { return "Proj" }
func (Projects) Hotkey() string  { return "ctrl+o" }
func (p Projects) Enabled() bool { return p.cfg.Enabled }

func (p Projects) Init() tea.Cmd {
	if !p.cfg.Enabled {
		return nil
	}

	dirs := p.cfg.Dirs
	singles := p.cfg.Projects

	return func() tea.Msg {
		var projects []project

		seen := map[string]bool{}

		add := func(path string) {
			path = filepath.Clean(path)

			if seen[path] {
				return
			}

			seen[path] = true

			_, statErr := os.Stat(filepath.Join(path, ".git"))

			projects = append(projects, project{
				name: filepath.Base(path),
				path: path,
				git:  statErr == nil,
			})
		}

		for _, dir := range dirs {
			base := expandHome(dir)

			entries, err := os.ReadDir(base)

			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
					add(filepath.Join(base, entry.Name()))
				}
			}
		}

		for _, single := range singles {
			path := expandHome(single)

			if info, err := os.Stat(path); err == nil && info.IsDir() {
				add(path)
			}
		}

		sort.Slice(projects, func(i, j int) bool {
			return strings.ToLower(projects[i].name) < strings.ToLower(projects[j].name)
		})

		duplicates := map[string]int{}

		for _, proj := range projects {
			duplicates[proj.name]++
		}

		for i := range projects {
			if duplicates[projects[i].name] > 1 {
				projects[i].label = collapseHome(projects[i].path)
			} else {
				projects[i].label = projects[i].name
			}
		}

		return projectsLoadedMsg(projects)
	}
}

func (p Projects) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		p.list.setItems(msg)

		var cmds []tea.Cmd

		for _, item := range p.list.items {
			if !item.git {
				continue
			}

			p.pending++

			path := item.path

			cmds = append(cmds, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), projectFetchTimeout)

				defer cancel()

				fetch := exec.CommandContext(ctx, "git", "-C", path, "fetch", "--quiet")
				fetch.Env = append(os.Environ(),
					"GIT_TERMINAL_PROMPT=0",
					"GIT_ASKPASS=true",
					"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
				)

				fetchFailed := fetch.Run() != nil

				status := gitStatus{}

				if output, err := exec.Command("git", "-C", path, "status", "--porcelain=v2", "--branch").Output(); err == nil {
					status = parseGitStatus(string(output))
				}

				return projectStatusMsg{
					path:        path,
					status:      status,
					fetchFailed: fetchFailed,
				}
			})
		}

		if p.pending > 0 {
			cmds = append(cmds, projectsTickCmd())
		}

		return p, tea.Batch(cmds...)

	case projectStatusMsg:
		items := make([]project, len(p.list.items))
		copy(items, p.list.items)

		for i := range items {
			if items[i].path == msg.path {
				items[i].statusKnown = true
				items[i].branch = msg.status.branch
				items[i].dirty = msg.status.dirty
				items[i].ahead = msg.status.ahead
				items[i].behind = msg.status.behind
				items[i].fetchFailed = msg.fetchFailed
			}
		}

		p.list.setItems(items)

		if p.pending > 0 {
			p.pending--
		}

		return p, nil

	case projectsTickMsg:
		if p.pending == 0 {
			return p, nil
		}

		p.frame++

		return p, projectsTickCmd()

	case editorMissingMsg:
		p.errorText = "no editor found — set editor in [projects] config or $EDITOR"

		return p, nil
	}

	return p, nil
}

func projectsTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return projectsTickMsg{}
	})
}

func parseGitStatus(output string) gitStatus {
	var status gitStatus

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			status.branch = strings.TrimPrefix(line, "# branch.head ")

		case strings.HasPrefix(line, "# branch.ab "):
			fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))

			if len(fields) == 2 {
				status.ahead, _ = strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
				status.behind, _ = strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
			}

		case line != "" && !strings.HasPrefix(line, "#"):
			status.dirty = true
		}
	}

	return status
}

func (p Projects) SetQuery(query string) Mode {
	p.errorText = ""
	p.list.setQuery(query)

	return p
}

func (Projects) Accent() lipgloss.Color { return projectsAccent }

func (p Projects) Status() string {
	switch {
	case !p.list.loaded:
		return subtleStyle.Render("scanning projects…")
	case len(p.list.items) == 0:
		return subtleStyle.Render("no projects found")
	case len(p.list.filtered) == 0:
		return subtleStyle.Render("no matching projects")
	case p.errorText != "":
		return errorStyle.Render(p.errorText)
	}

	return ""
}

func (p Projects) Rows() []Row {
	frame := projectSpinnerFrames[p.frame%len(projectSpinnerFrames)]

	return p.list.rows(func(item project) Row {
		right := ""

		switch {
		case !item.git:

		case !item.statusKnown:
			right = subtleStyle.Render(frame)

		default:
			branch := item.branch

			if branch == "" {
				branch = "?"
			}

			branchStyle := cleanBranchStyle

			if item.dirty {
				branchStyle = dirtyBranchStyle
			}

			arrows := ""

			if item.ahead > 0 {
				arrows += "↑"
			}

			if item.behind > 0 {
				arrows += "↓"
			}

			prefix := ""

			if arrows != "" {
				prefix = aheadBehindStyle.Render(arrows)
			}

			if item.fetchFailed {
				prefix += "!"
			}

			right = branchStyle.Render(branch)

			if prefix != "" {
				right = prefix + " " + right
			}
		}

		return Row{left: item.label, right: right}
	})
}

func (p Projects) Activate(index int) tea.Cmd {
	item, ok := p.list.at(index)

	if !ok {
		return nil
	}

	editor := p.cfg.Editor

	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	if editor == "" {
		return func() tea.Msg {
			return editorMissingMsg{}
		}
	}

	terminal := p.cfg.Terminal
	editorTerminal := p.cfg.EditorTerminal

	return func() tea.Msg {
		argv := strings.Fields(editor)

		if directoryCapableEditors[filepath.Base(argv[0])] {
			argv = append(argv, ".")
		}

		inTerminal := terminalEditors[filepath.Base(argv[0])]

		if editorTerminal != nil {
			inTerminal = *editorTerminal
		}

		if inTerminal {
			argv = terminalArgv(resolveTerminal(terminal), strings.Join(argv, " "))
		}

		spawnDetached(item.path, argv...)

		return RequestQuitMsg{}
	}
}

var terminalEditors = map[string]bool{
	"hx": true, "helix": true, "vi": true, "vim": true, "nvim": true,
	"nano": true, "micro": true, "kak": true, "kakoune": true,
}

var directoryCapableEditors = map[string]bool{
	"hx": true, "helix": true, "vi": true, "vim": true, "nvim": true,
	"emacs": true, "code": true, "codium": true, "subl": true, "zed": true,
}
