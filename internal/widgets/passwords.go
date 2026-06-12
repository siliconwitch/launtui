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
)

type PasswordsConfig struct {
	Enabled bool   `toml:"enabled"`
	Store   string `toml:"store"`
}

func (PasswordsConfig) SectionName() string { return "passwords" }

func DefaultPasswordsConfig() PasswordsConfig {
	return PasswordsConfig{Enabled: true}
}

var passwordsAccent = lipgloss.Color("3")

type passwordEntriesMsg []string

type passwordShownMsg struct {
	output string
	err    error
}

type passwordCopyBlockedMsg struct{}

type Passwords struct {
	cfg       PasswordsConfig
	list      list[string]
	errorText string
}

func NewPasswords(cfg PasswordsConfig) Passwords {
	return Passwords{cfg: cfg, list: newList(func(entry string) string { return entry })}
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
		p.list.setItems(msg)

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
	p.errorText = ""
	p.list.setQuery(query)

	return p
}

func (p Passwords) HasResults() bool { return p.list.hasResults() }

func (p Passwords) MoveUp() Mode {
	p.list.moveUp()

	return p
}

func (p Passwords) MoveDown() Mode {
	p.list.moveDown()

	return p
}

func (p Passwords) Activate() tea.Cmd {
	entry, ok := p.list.selected()

	if !ok {
		return nil
	}

	return showPasswordCmd(entry)
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
	case !p.list.loaded:
		return subtleStyle.Render("scanning password store…")
	case len(p.list.items) == 0:
		return subtleStyle.Render("no password store found")
	case len(p.list.filtered) == 0:
		return subtleStyle.Render("no matching passwords")
	}

	if p.errorText != "" {
		return errorStyle.Render(p.errorText) + "\n" + p.list.view(width, rows-1, p.renderEntry)
	}

	return p.list.view(width, rows, p.renderEntry)
}

func (p Passwords) renderEntry(entry string, selected bool, width int) string {
	return renderRow(passwordsAccent, selected, truncate(entry, max(width-2, 1)), "")
}
