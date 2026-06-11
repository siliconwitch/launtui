package widgets

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type OnePasswordConfig struct {
	Enabled bool   `toml:"enabled"`
	Account string `toml:"account"`
}

func (OnePasswordConfig) SectionName() string { return "onepassword" }

func DefaultOnePasswordConfig() OnePasswordConfig {
	return OnePasswordConfig{Enabled: true}
}

var (
	onePasswordSelectedStyle = lipgloss.NewStyle().Foreground(onepasswordColor).Bold(true)
	onePasswordBarStyle      = lipgloss.NewStyle().Foreground(onepasswordColor)
)

var errOnePasswordNoPassword = errors.New("no password field")

type onePasswordItem struct {
	id       string
	account  string
	title    string
	subtitle string
	username string
}

type onePasswordItemsMsg struct {
	items []onePasswordItem
	err   error
}

type onePasswordCopiedMsg struct {
	err     error
	blocked bool
}

type OnePassword struct {
	cfg       OnePasswordConfig
	available bool
	items     []onePasswordItem
	filtered  []onePasswordItem
	query     string
	cursor    int
	loaded    bool
	errorText string
}

func NewOnePassword(cfg OnePasswordConfig) OnePassword {
	_, err := exec.LookPath("op")

	return OnePassword{cfg: cfg, available: err == nil}
}

func (OnePassword) Name() string   { return "1Pass" }
func (OnePassword) Hotkey() string { return "ctrl+1" }

func (p OnePassword) Enabled() bool { return p.cfg.Enabled && p.available }

func (p OnePassword) Init() tea.Cmd {
	if !p.Enabled() {
		return nil
	}

	return loadOnePasswordItemsCmd(p.cfg)
}

func loadOnePasswordItemsCmd(cfg OnePasswordConfig) tea.Cmd {
	return func() tea.Msg {
		items, err := loadOnePasswordItems(cfg)

		return onePasswordItemsMsg{items: items, err: err}
	}
}

type opAccount struct {
	id    string
	label string
}

func loadOnePasswordItems(cfg OnePasswordConfig) ([]onePasswordItem, error) {
	accounts, err := onePasswordAccounts(cfg.Account)

	if err != nil {
		return nil, err
	}

	multiple := len(accounts) > 1

	var items []onePasswordItem
	var firstErr error

	for _, account := range accounts {
		got, err := onePasswordItemsForAccount(account, multiple)

		if err != nil {
			if firstErr == nil {
				firstErr = err
			}

			continue
		}

		items = append(items, got...)
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].title) < strings.ToLower(items[j].title)
	})

	if len(items) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return items, nil
}

func onePasswordAccounts(configured string) ([]opAccount, error) {
	if configured != "" {
		return []opAccount{{id: configured, label: configured}}, nil
	}

	output, err := opCommand("", "account", "list", "--format=json").Output()

	if err != nil {
		return nil, err
	}

	return parseOnePasswordAccounts(output)
}

func parseOnePasswordAccounts(data []byte) ([]opAccount, error) {
	var raw []struct {
		URL         string `json:"url"`
		Email       string `json:"email"`
		AccountUUID string `json:"account_uuid"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	accounts := make([]opAccount, 0, len(raw))

	for _, entry := range raw {
		label := entry.Email

		if label == "" {
			label = entry.URL
		}

		accounts = append(accounts, opAccount{id: entry.AccountUUID, label: label})
	}

	return accounts, nil
}

func onePasswordItemsForAccount(account opAccount, withAccountLabel bool) ([]onePasswordItem, error) {
	output, err := opCommand(account.id, "item", "list", "--format=json", "--categories", "Login,Password").Output()

	if err != nil {
		return nil, err
	}

	return parseOnePasswordItems(output, account, withAccountLabel)
}

func parseOnePasswordItems(data []byte, account opAccount, withAccountLabel bool) ([]onePasswordItem, error) {
	var raw []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Vault struct {
			Name string `json:"name"`
		} `json:"vault"`
		AdditionalInformation string `json:"additional_information"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	items := make([]onePasswordItem, 0, len(raw))

	for _, entry := range raw {
		username := strings.TrimSpace(entry.AdditionalInformation)

		if username == "—" {
			username = ""
		}

		parts := make([]string, 0, 2)

		if username != "" {
			parts = append(parts, username)
		}

		if withAccountLabel {
			parts = append(parts, account.label)
		} else if entry.Vault.Name != "" {
			parts = append(parts, entry.Vault.Name)
		}

		items = append(items, onePasswordItem{
			id:       entry.ID,
			account:  account.id,
			title:    entry.Title,
			subtitle: strings.Join(parts, " · "),
			username: username,
		})
	}

	return items, nil
}

func opCommand(account string, args ...string) *exec.Cmd {
	if account != "" {
		args = append([]string{"--account", account}, args...)
	}

	cmd := exec.Command("op", args...)
	cmd.Env = append(os.Environ(), "OP_BIOMETRIC_UNLOCK_ENABLED=true")

	return cmd
}

func (p OnePassword) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch msg := msg.(type) {
	case onePasswordItemsMsg:
		p.loaded = true

		if msg.err != nil {
			p.errorText = "1Password unavailable — is `op` signed in?"

			return p, nil
		}

		p.items = msg.items
		p.refilter()

		return p, nil

	case onePasswordCopiedMsg:
		if msg.blocked {
			p.errorText = "could not protect clipboard history — password not copied"
		} else if msg.err != nil {
			p.errorText = "could not read that item's password"
		}

		return p, nil
	}

	return p, nil
}

func (p OnePassword) SetQuery(query string) Mode {
	p.query = query
	p.cursor = 0
	p.errorText = ""
	p.refilter()

	return p
}

func (p *OnePassword) refilter() {
	query := strings.TrimSpace(p.query)

	if query == "" {
		p.filtered = p.items
	} else {
		titles := make([]string, len(p.items))

		for i, item := range p.items {
			titles[i] = item.title
		}

		matches := fuzzy.Find(query, titles)
		p.filtered = make([]onePasswordItem, len(matches))

		for i, match := range matches {
			p.filtered[i] = p.items[match.Index]
		}
	}

	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p OnePassword) HasResults() bool {
	return p.loaded && len(p.filtered) > 0
}

func (p OnePassword) MoveUp() Mode {
	if p.cursor > 0 {
		p.cursor--
	}

	return p
}

func (p OnePassword) MoveDown() Mode {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++
	}

	return p
}

func (p OnePassword) Activate() tea.Cmd {
	if len(p.filtered) == 0 {
		return nil
	}

	return revealOnePasswordCmd(p.filtered[p.cursor])
}

func revealOnePasswordCmd(item onePasswordItem) tea.Cmd {
	return func() tea.Msg {
		output, err := opCommand(item.account, "item", "get", item.id, "--reveal", "--fields", "label=password").Output()

		if err != nil {
			return onePasswordCopiedMsg{err: err}
		}

		password := strings.TrimSpace(string(output))

		if password == "" {
			return onePasswordCopiedMsg{err: errOnePasswordNoPassword}
		}

		if suppressClipboardRecording(password) != nil {
			return onePasswordCopiedMsg{blocked: true}
		}

		copyToClipboard(password)

		if item.username != "" {
			recordClipboardText(item.username, 0)
		}

		return RequestQuitMsg{}
	}
}

func (p OnePassword) View(width, rows int) string {
	if !p.loaded {
		return subtleStyle.Render("unlocking 1Password…")
	}

	var lines []string

	if p.errorText != "" {
		lines = append(lines, errorStyle.Render(p.errorText))
		rows--
	}

	switch {
	case len(p.items) == 0:
		if p.errorText == "" {
			lines = append(lines, subtleStyle.Render("no 1Password logins found"))
		}

		return strings.Join(lines, "\n")

	case len(p.filtered) == 0:
		lines = append(lines, subtleStyle.Render("no matching items"))

		return strings.Join(lines, "\n")
	}

	start, end := visibleRange(p.cursor, rows, len(p.filtered))

	for i := start; i < end; i++ {
		lines = append(lines, p.renderItem(p.filtered[i], i == p.cursor, width))
	}

	return strings.Join(lines, "\n")
}

func (p OnePassword) renderItem(item onePasswordItem, selected bool, width int) string {
	avail := width - 2

	if avail < 1 {
		avail = 1
	}

	title, subtitle := item.title, item.subtitle

	if displayWidth(title) > avail {
		title = truncate(title, avail)
		subtitle = ""
	}

	sub := ""

	if subtitle != "" {
		if gap := avail - displayWidth(title); gap > 3 {
			subtitle = truncate(subtitle, gap-2)
			sub = strings.Repeat(" ", gap-displayWidth(subtitle)) + subtleStyle.Render(subtitle)
		}
	}

	if selected {
		return onePasswordBarStyle.Render("▌ ") + onePasswordSelectedStyle.Render(title) + sub
	}

	return "  " + nameStyle.Render(title) + sub
}
