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

type passwordUsernameMsg struct {
	entry    string
	username string
}

type Passwords struct {
	cfg       PasswordsConfig
	list      list[string]
	usernames map[string]string
	selected  string
	errorText string
}

func NewPasswords(cfg PasswordsConfig) Passwords {
	return Passwords{
		cfg:       cfg,
		list:      newList(func(entry string) string { return entry }),
		usernames: map[string]string{},
	}
}

func (Passwords) Name() string    { return "Pass" }
func (Passwords) Hotkey() string  { return "ctrl+p" }
func (p Passwords) Enabled() bool { return p.cfg.Enabled }

func (p Passwords) Init() tea.Cmd {
	if !p.cfg.Enabled {
		return nil
	}

	return func() tea.Msg {
		store := ""

		if p.cfg.Store != "" {
			store = expandHome(p.cfg.Store)
		} else if dir := os.Getenv("PASSWORD_STORE_DIR"); dir != "" {
			store = dir
		} else if home, err := os.UserHomeDir(); err == nil {
			store = filepath.Join(home, ".password-store")
		}

		if store == "" {
			return passwordEntriesMsg(nil)
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

		return passwordEntriesMsg(entries)
	}
}

func (p Passwords) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch msg := msg.(type) {
	case passwordEntriesMsg:
		p.list.setItems(msg)

		return p, nil

	case passwordUsernameMsg:
		p.usernames[msg.entry] = msg.username

		return p, nil

	case passwordShownMsg:
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

	case passwordCopyBlockedMsg:
		p.errorText = "could not protect clipboard history — password not copied"

		return p, nil
	}

	return p, nil
}

func (p Passwords) SetQuery(query string) Mode {
	p.errorText = ""
	p.list.setQuery(query)

	return p
}

func (Passwords) Accent() lipgloss.Color { return passwordsAccent }

func (p Passwords) Status() string {
	switch {
	case !p.list.loaded:
		return subtleStyle.Render("scanning password store…")
	case len(p.list.items) == 0:
		return subtleStyle.Render("no password store found")
	case len(p.list.filtered) == 0:
		return subtleStyle.Render("no matching passwords")
	case p.errorText != "":
		return errorStyle.Render(p.errorText)
	}

	return ""
}

func (p Passwords) Rows() []Row {
	return p.list.rows(func(entry string) Row {
		right := ""

		if entry == p.selected {
			if username := p.usernames[entry]; username != "" {
				right = subtleStyle.Render(username)
			}
		}

		return Row{left: entry, right: right}
	})
}

func (p Passwords) Select(index int) (Mode, tea.Cmd) {
	entry, ok := p.list.at(index)

	if !ok {
		p.selected = ""

		return p, nil
	}

	p.selected = entry

	if _, known := p.usernames[entry]; known {
		return p, nil
	}

	store := p.cfg.Store

	return p, func() tea.Msg {
		cmd := exec.Command("pass", "show", entry)
		cmd.Env = append(os.Environ(), "PASSWORD_STORE_GPG_OPTS=--pinentry-mode cancel")

		if store != "" {
			cmd.Env = append(cmd.Env, "PASSWORD_STORE_DIR="+expandHome(store))
		}

		output, err := cmd.Output()

		if err != nil {
			return nil
		}

		lines := strings.Split(string(output), "\n")

		if len(lines) < 2 {
			return nil
		}

		return passwordUsernameMsg{entry: entry, username: strings.TrimSpace(lines[1])}
	}
}

func (p Passwords) Activate(index int) tea.Cmd {
	entry, ok := p.list.at(index)

	if !ok {
		return nil
	}

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
