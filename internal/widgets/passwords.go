package widgets

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type PasswordsConfig struct {
	Enabled bool   `toml:"enabled"`
	Store   string `toml:"store"`
}

func (PasswordsConfig) SectionName() string { return "passwords" }

func DefaultPasswordsConfig() PasswordsConfig {
	return PasswordsConfig{Enabled: true}
}

var (
	passwordSelectedStyle = lipgloss.NewStyle().Foreground(passwordColor).Bold(true)
	passwordBarStyle      = lipgloss.NewStyle().Foreground(passwordColor)
)

type passwordEntriesMsg []string

type passwordShownMsg struct {
	output string
	err    error
}

type passwordCopyBlockedMsg struct{}

type Passwords struct {
	cfg       PasswordsConfig
	entries   []string
	filtered  []string
	query     string
	cursor    int
	loaded    bool
	errorText string
}

func NewPasswords(cfg PasswordsConfig) Passwords {
	return Passwords{cfg: cfg}
}

func (Passwords) Name() string    { return "Pass" }
func (Passwords) Hotkey() string  { return "ctrl+p" }
func (p Passwords) Enabled() bool { return p.cfg.Enabled }

func (p Passwords) Init() tea.Cmd {
	if !p.cfg.Enabled {
		return nil
	}

	return loadPasswordEntriesCmd(p.cfg.Store)
}

func passwordStoreDir(configured string) string {
	if configured != "" {
		return expandHome(configured)
	}

	if dir := os.Getenv("PASSWORD_STORE_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()

	if err != nil {
		return ""
	}

	return filepath.Join(home, ".password-store")
}

func loadPasswordEntriesCmd(configuredStore string) tea.Cmd {
	return func() tea.Msg {
		return passwordEntriesMsg(scanPasswordStore(passwordStoreDir(configuredStore)))
	}
}

func scanPasswordStore(store string) []string {
	if store == "" {
		return nil
	}

	var entries []string

	_ = filepath.WalkDir(store, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") && path != store {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(entry.Name(), ".gpg") {
			return nil
		}

		relative, err := filepath.Rel(store, path)

		if err != nil {
			return nil
		}

		entries = append(entries, strings.TrimSuffix(relative, ".gpg"))

		return nil
	})

	sort.Strings(entries)

	return entries
}

func (p Passwords) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch msg := msg.(type) {
	case passwordEntriesMsg:
		p.entries = msg
		p.loaded = true
		p.refilter()

		return p, nil

	case passwordShownMsg:
		return p.handleShown(msg)

	case passwordCopyBlockedMsg:
		p.errorText = "could not protect clipboard history — password not copied"

		return p, nil
	}

	return p, nil
}

func (p Passwords) handleShown(msg passwordShownMsg) (Mode, tea.Cmd) {
	if msg.err != nil {
		p.errorText = "pass failed — wrong passphrase or cancelled"

		return p, nil
	}

	lines := strings.Split(msg.output, "\n")
	password := strings.TrimRight(lines[0], "\r")

	if password == "" {
		p.errorText = "entry is empty"

		return p, nil
	}

	username := ""

	if len(lines) > 1 {
		username = strings.TrimSpace(lines[1])
	}

	return p, func() tea.Msg {
		if suppressClipboardRecording(password) != nil {
			return passwordCopyBlockedMsg{}
		}

		copyToClipboard(password)

		if username != "" {
			recordClipboardText(username, 0)
		}

		return RequestQuitMsg{}
	}
}

func (p Passwords) SetQuery(query string) Mode {
	p.query = query
	p.cursor = 0
	p.errorText = ""
	p.refilter()

	return p
}

func (p *Passwords) refilter() {
	query := strings.TrimSpace(p.query)

	if query == "" {
		p.filtered = p.entries
	} else {
		matches := fuzzy.Find(query, p.entries)
		p.filtered = make([]string, len(matches))

		for i, match := range matches {
			p.filtered[i] = p.entries[match.Index]
		}
	}

	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p Passwords) HasResults() bool {
	return p.loaded && len(p.filtered) > 0
}

func (p Passwords) MoveUp() Mode {
	if p.cursor > 0 {
		p.cursor--
	}

	return p
}

func (p Passwords) MoveDown() Mode {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++
	}

	return p
}

func (p Passwords) Activate() tea.Cmd {
	if len(p.filtered) == 0 {
		return nil
	}

	return showPasswordCmd(p.filtered[p.cursor])
}

func showPasswordCmd(entry string) tea.Cmd {
	cmd := exec.Command("pass", "show", entry)

	var output bytes.Buffer
	cmd.Stdout = &output

	if os.Getenv("GPG_TTY") == "" {
		tty, err := os.Readlink("/proc/self/fd/0")

		if err != nil {
			tty = "/dev/tty"
		}

		cmd.Env = append(os.Environ(), "GPG_TTY="+tty)
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return passwordShownMsg{output: output.String(), err: err}
	})
}

func (p Passwords) View(width, rows int) string {
	switch {
	case !p.loaded:
		return subtleStyle.Render("scanning password store…")
	case len(p.entries) == 0:
		return subtleStyle.Render("no password store found")
	case len(p.filtered) == 0:
		return subtleStyle.Render("no matching passwords")
	}

	var lines []string

	if p.errorText != "" {
		lines = append(lines, errorStyle.Render(p.errorText))
		rows--
	}

	start, end := visibleRange(p.cursor, rows, len(p.filtered))

	for i := start; i < end; i++ {
		lines = append(lines, p.renderEntry(p.filtered[i], i == p.cursor, width))
	}

	return strings.Join(lines, "\n")
}

func (p Passwords) renderEntry(entry string, selected bool, width int) string {
	name := truncate(entry, max(width-2, 1))

	if selected {
		return passwordBarStyle.Render("▌ ") + passwordSelectedStyle.Render(name)
	}

	return "  " + nameStyle.Render(name)
}
