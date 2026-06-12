package widgets

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProjectsConfig struct {
	Enabled bool   `toml:"enabled"`
	Dir     string `toml:"dir"`
	Editor  string `toml:"editor"`
}

func (ProjectsConfig) SectionName() string { return "projects" }

func DefaultProjectsConfig() ProjectsConfig {
	return ProjectsConfig{Enabled: true, Dir: "~/projects"}
}

const projectFetchTimeout = 10 * time.Second

var (
	projectsAccent   = lipgloss.Color("2")
	cleanBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	dirtyBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	aheadBehindStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	projectSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	errNoEditor = errors.New("no editor configured")
)

type project struct {
	name        string
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
	name        string
	status      gitStatus
	fetchFailed bool
}

type projectsTickMsg struct{}

type editorDoneMsg struct {
	err error
}

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

	return scanProjectsCmd(p.cfg.Dir)
}

func scanProjectsCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		return projectsLoadedMsg(scanProjects(expandHome(dir)))
	}
}

func scanProjects(dir string) []project {
	entries, err := os.ReadDir(dir)

	if err != nil {
		return nil
	}

	var projects []project

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		_, statErr := os.Stat(filepath.Join(path, ".git"))

		projects = append(projects, project{
			name: entry.Name(),
			path: path,
			git:  statErr == nil,
		})
	}

	return projects
}

func (p Projects) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		return p.handleLoaded(msg)

	case projectStatusMsg:
		return p.handleStatus(msg)

	case projectsTickMsg:
		if p.pending == 0 {
			return p, nil
		}

		p.frame++

		return p, projectsTickCmd()

	case editorDoneMsg:
		return p.handleEditorDone(msg)
	}

	return p, nil
}

func (p Projects) handleLoaded(msg projectsLoadedMsg) (Mode, tea.Cmd) {
	p.list.setItems(msg)

	var cmds []tea.Cmd

	for _, item := range p.list.items {
		if item.git {
			p.pending++

			cmds = append(cmds, projectStatusCmd(item.name, item.path))
		}
	}

	if p.pending > 0 {
		cmds = append(cmds, projectsTickCmd())
	}

	return p, tea.Batch(cmds...)
}

func (p Projects) handleStatus(msg projectStatusMsg) (Mode, tea.Cmd) {
	items := make([]project, len(p.list.items))
	copy(items, p.list.items)

	for i := range items {
		if items[i].name == msg.name {
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
}

func (p Projects) handleEditorDone(msg editorDoneMsg) (Mode, tea.Cmd) {
	if errors.Is(msg.err, errNoEditor) {
		p.errorText = "no editor found — set editor in [projects] config or $EDITOR"

		return p, nil
	}

	if msg.err != nil {
		p.errorText = "editor exited with an error"

		return p, nil
	}

	return p, func() tea.Msg {
		return RequestQuitMsg{}
	}
}

func projectsTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return projectsTickMsg{}
	})
}

func projectStatusCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		fetchFailed := fetchProject(path) != nil

		return projectStatusMsg{
			name:        name,
			status:      readProjectStatus(path),
			fetchFailed: fetchFailed,
		}
	}
}

func fetchProject(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), projectFetchTimeout)

	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", path, "fetch", "--quiet")
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=true",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
	)

	return cmd.Run()
}

func readProjectStatus(path string) gitStatus {
	output, err := exec.Command("git", "-C", path, "status", "--porcelain=v2", "--branch").Output()

	if err != nil {
		return gitStatus{}
	}

	return parseGitStatus(string(output))
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

func (p Projects) HasResults() bool { return p.list.hasResults() }

func (p Projects) MoveUp() Mode {
	p.list.moveUp()

	return p
}

func (p Projects) MoveDown() Mode {
	p.list.moveDown()

	return p
}

func (p Projects) Activate() tea.Cmd {
	item, ok := p.list.selected()

	if !ok {
		return nil
	}

	editor := resolveEditor(p.cfg.Editor)

	if editor == "" {
		return func() tea.Msg {
			return editorDoneMsg{err: errNoEditor}
		}
	}

	words := editorArgv(editor)

	cmd := exec.Command(words[0], words[1:]...)
	cmd.Dir = item.path

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorDoneMsg{err: err}
	})
}

func resolveEditor(configured string) string {
	if configured != "" {
		return configured
	}

	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	return os.Getenv("EDITOR")
}

var directoryCapableEditors = map[string]bool{
	"hx": true, "helix": true, "vi": true, "vim": true, "nvim": true,
	"emacs": true, "code": true, "codium": true, "subl": true, "zed": true,
}

func editorArgv(editor string) []string {
	words := strings.Fields(editor)

	if directoryCapableEditors[filepath.Base(words[0])] {
		return append(words, ".")
	}

	return words
}

func (p Projects) View(width, rows int) string {
	switch {
	case !p.list.loaded:
		return subtleStyle.Render("scanning projects…")
	case len(p.list.items) == 0:
		return subtleStyle.Render("no projects in " + p.cfg.Dir)
	case len(p.list.filtered) == 0:
		return subtleStyle.Render("no matching projects")
	}

	if p.errorText != "" {
		return errorStyle.Render(p.errorText) + "\n" + p.list.view(width, rows-1, p.renderProject)
	}

	return p.list.view(width, rows, p.renderProject)
}

func (p Projects) renderProject(item project, selected bool, width int) string {
	avail := max(width-2, 1)
	name := truncate(item.name, avail)

	status, statusWidth := p.projectStatus(item)

	sub := ""

	if status != "" {
		if gap := avail - lipgloss.Width(name); gap > statusWidth+1 {
			sub = strings.Repeat(" ", gap-statusWidth) + status
		}
	}

	return renderRow(projectsAccent, selected, name, sub)
}

func (p Projects) projectStatus(item project) (string, int) {
	if !item.git {
		return "", 0
	}

	if !item.statusKnown {
		frame := projectSpinnerFrames[p.frame%len(projectSpinnerFrames)]

		return subtleStyle.Render(frame), lipgloss.Width(frame)
	}

	branch := item.branch

	if branch == "" {
		branch = "?"
	}

	branchStyle := cleanBranchStyle

	if item.dirty {
		branchStyle = dirtyBranchStyle
	}

	styled := branchStyle.Render(branch)
	statusWidth := lipgloss.Width(branch)

	arrows := ""

	if item.ahead > 0 {
		arrows += "↑"
	}

	if item.behind > 0 {
		arrows += "↓"
	}

	prefix := ""
	prefixWidth := 0

	if arrows != "" {
		prefix = aheadBehindStyle.Render(arrows)
		prefixWidth = lipgloss.Width(arrows)
	}

	if item.fetchFailed {
		prefix += "!"
		prefixWidth++
	}

	if prefix != "" {
		styled = prefix + " " + styled
		statusWidth += prefixWidth + 1
	}

	return styled, statusWidth
}
