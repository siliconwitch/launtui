package widgets

import (
	"bytes"
	"os"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	config    PasswordsConfig
	list      list[string]
	usernames map[string]string
	selected  string
	errorText string
}

func NewPasswords(config PasswordsConfig) Passwords {
	return Passwords{
		config:    config,
		list:      newList(func(entry string) string { return entry }),
		usernames: map[string]string{},
	}
}

func (Passwords) Name() string    { return "Pass" }
func (Passwords) Hotkey() string  { return "ctrl+p" }
func (p Passwords) Enabled() bool { return p.config.Enabled }

func (p Passwords) Init() tea.Cmd {
	if !p.config.Enabled {
		return nil
	}

	store := p.config.Store

	return func() tea.Msg {
		command := exec.Command("pass", "ls")

		if store != "" {
			command.Env = append(os.Environ(), "PASSWORD_STORE_DIR="+expandHome(store))
		}

		output, err := command.Output()

		if err != nil {
			return passwordEntriesMsg(nil)
		}

		type treeNode struct {
			depth int
			name  string
		}

		var nodes []treeNode

		for _, raw := range strings.Split(string(output), "\n") {
			runes := []rune(strings.ReplaceAll(ansi.Strip(raw), "\u00a0", " "))

			connector := -1

			for index, glyph := range runes {
				if glyph == '├' || glyph == '└' {
					connector = index

					break
				}
			}

			if connector < 0 {
				continue
			}

			name := strings.TrimSpace(strings.TrimLeft(string(runes[connector+1:]), "─ "))

			if name == "" {
				continue
			}

			nodes = append(nodes, treeNode{depth: connector / 4, name: name})
		}

		var entries []string
		var ancestors []string

		for index, node := range nodes {
			if node.depth > len(ancestors) {
				continue
			}

			ancestors = append(ancestors[:node.depth], node.name)

			isFolder := index+1 < len(nodes) && nodes[index+1].depth > node.depth

			if !isFolder {
				entries = append(entries, strings.Join(ancestors, "/"))
			}
		}

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

	store := p.config.Store

	return p, func() tea.Msg {
		command := exec.Command("pass", "show", entry)
		command.Env = append(os.Environ(), "PASSWORD_STORE_GPG_OPTS=--pinentry-mode cancel")

		if store != "" {
			command.Env = append(command.Env, "PASSWORD_STORE_DIR="+expandHome(store))
		}

		output, err := command.Output()

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

	command := exec.Command("pass", "show", entry)

	var output bytes.Buffer
	command.Stdout = &output

	if os.Getenv("GPG_TTY") == "" {
		tty, err := os.Readlink("/proc/self/fd/0")

		if err != nil {
			tty = "/dev/tty"
		}

		command.Env = append(os.Environ(), "GPG_TTY="+tty)
	}

	return tea.ExecProcess(command, func(err error) tea.Msg {
		return passwordShownMsg{output: output.String(), err: err}
	})
}
