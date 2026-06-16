package widgets

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

type Mode interface {
	Name() string
	Hotkey() string
	Enabled() bool
	Init() tea.Cmd
	Update(tea.Msg) (Mode, tea.Cmd)
	SetQuery(query string) Mode
	Rows() []Row
	Accent() lipgloss.Color
	Activate(index int) tea.Cmd
	Status() string
}

type RequestQuitMsg struct{}

type AppClosingMsg struct{}

type StrongMatcher interface {
	StrongMatch() bool
}

type RowDeleter interface {
	DeleteRow(index int) (Mode, tea.Cmd)
	ClearRows() (Mode, tea.Cmd)
}

type Selectable interface {
	Select(index int) (Mode, tea.Cmd)
}

type Recaller interface {
	RecallText(index int) (string, bool)
}

var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

type rowStyle int

const (
	rowPlain rowStyle = iota
	rowDim
	rowEmphasized
)

type Row struct {
	left      string
	right     string
	style     rowStyle
	Deletable bool
}

func HasResults(rows []Row) bool {
	for _, row := range rows {
		if row.style != rowDim {
			return true
		}
	}

	return false
}

func RenderResults(status string, rows []Row, accent lipgloss.Color, cursor, width, height int) string {
	var lines []string

	if status != "" {
		lines = append(lines, status)

		height -= lipgloss.Height(status)
	}

	if len(rows) > 0 && height > 0 {
		start, end := visibleRange(cursor, height, len(rows))

		for i := start; i < end; i++ {
			lines = append(lines, renderRow(accent, i == cursor, rows[i], width))
		}
	}

	return strings.Join(lines, "\n")
}

func renderRow(accent lipgloss.Color, selected bool, row Row, width int) string {
	availableWidth := max(width-2, 1)
	name := row.left
	rightColumn := ""

	if lipgloss.Width(name) > availableWidth {
		name = truncate(name, availableWidth)
	} else if row.right != "" {
		gap := availableWidth - lipgloss.Width(name)

		if gap > 2 {
			right := truncate(row.right, gap-1)
			rightColumn = strings.Repeat(" ", gap-lipgloss.Width(right)) + right
		}
	}

	if selected {
		accentStyle := lipgloss.NewStyle().Foreground(accent)

		return accentStyle.Render("▌ ") + accentStyle.Bold(true).Render(name) + rightColumn
	}

	switch row.style {
	case rowDim:
		return "  " + subtleStyle.Render(name) + rightColumn
	case rowEmphasized:
		return "  " + lipgloss.NewStyle().Foreground(accent).Bold(true).Render(name) + rightColumn
	default:
		return "  " + name + rightColumn
	}
}

func truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= width {
		return text
	}

	if width == 1 {
		return "…"
	}

	return ansi.Truncate(text, width-1, "") + "…"
}

func relativeAge(elapsed int64) string {
	switch {
	case elapsed < 60:
		return "now"
	case elapsed < 3600:
		return strconv.FormatInt(elapsed/60, 10) + "m"
	case elapsed < 86400:
		return strconv.FormatInt(elapsed/3600, 10) + "h"
	default:
		return strconv.FormatInt(elapsed/86400, 10) + "d"
	}
}

func visibleRange(cursor, rows, count int) (int, int) {
	if rows < 1 {
		rows = 1
	}

	start := 0

	if cursor >= rows {
		start = cursor - rows + 1
	}

	return start, min(start+rows, count)
}

type list[T any] struct {
	key      func(T) string
	items    []T
	filtered []T
	query    string
	loaded   bool
}

func newList[T any](key func(T) string) list[T] {
	return list[T]{key: key}
}

func (l *list[T]) setItems(items []T) {
	l.items = items
	l.loaded = true
	l.refilter()
}

func (l *list[T]) setQuery(query string) {
	l.query = query
	l.refilter()
}

func (l *list[T]) refilter() {
	query := strings.TrimSpace(l.query)

	if query == "" {
		l.filtered = l.items

		return
	}

	names := make([]string, len(l.items))

	for i, item := range l.items {
		names[i] = l.key(item)
	}

	matches := fuzzy.Find(query, names)
	l.filtered = make([]T, len(matches))

	for i, match := range matches {
		l.filtered[i] = l.items[match.Index]
	}
}

func (l list[T]) rows(cell func(item T) Row) []Row {
	rows := make([]Row, len(l.filtered))

	for i, item := range l.filtered {
		rows[i] = cell(item)
	}

	return rows
}

func (l list[T]) at(index int) (T, bool) {
	if index < 0 || index >= len(l.filtered) {
		var zero T

		return zero, false
	}

	return l.filtered[index], true
}

func removeAt[T any](entries []T, index int) []T {
	return append(append([]T{}, entries[:index]...), entries[index+1:]...)
}

func prependCapped[T any](entries []T, entry T, limit int, duplicate func(T) bool) []T {
	kept := make([]T, 0, len(entries)+1)
	kept = append(kept, entry)

	for _, existing := range entries {
		if duplicate == nil || !duplicate(existing) {
			kept = append(kept, existing)
		}
	}

	if limit > 0 && len(kept) > limit {
		kept = kept[:limit]
	}

	return kept
}

func copyToClipboard(text string) {
	tools := [][]string{
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
	}

	for _, tool := range tools {
		path, err := exec.LookPath(tool[0])

		if err != nil {
			continue
		}

		command := exec.Command(path, tool[1:]...)
		command.Stdin = strings.NewReader(text)

		if command.Run() == nil {
			return
		}
	}
}

func readClipboard() string {
	tools := [][]string{
		{"wl-paste", "--no-newline", "--type", "text"},
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
	}

	for _, tool := range tools {
		path, err := exec.LookPath(tool[0])

		if err != nil {
			continue
		}

		output, err := exec.Command(path, tool[1:]...).Output()

		if err != nil {
			continue
		}

		return string(output)
	}

	return ""
}

const (
	clipboardHistoryFile = "clipboard-history.json"
	suppressionFile      = "suppressed.json"
)

type clipboardEntry struct {
	Text string `json:"text"`
	Time int64  `json:"time"`
}

func loadClipboardHistory() []clipboardEntry {
	path, err := launtuiDataPath(clipboardHistoryFile)

	if err != nil {
		return nil
	}

	entries, _ := loadJSON[[]clipboardEntry](path)

	return entries
}

func recordClipboardText(text string, limit int) []clipboardEntry {
	if strings.TrimSpace(text) == "" {
		return loadClipboardHistory()
	}

	suppressed := false

	if path, err := launtuiCachePath(suppressionFile); err == nil {
		suppression, ok := loadJSON[clipboardSuppression](path)

		switch {
		case !ok:
		case time.Now().Unix() > suppression.Expires:
			_ = os.Remove(path)
		case saltedHash(suppression.Salt, text) == suppression.Hash:
			suppressed = true
		}
	}

	if suppressed {
		return loadClipboardHistory()
	}

	entry := clipboardEntry{Text: text, Time: time.Now().Unix()}

	entries := prependCapped(loadClipboardHistory(), entry, limit, func(existing clipboardEntry) bool {
		return existing.Text == text
	})

	saveClipboardHistory(entries)

	return entries
}

func saveClipboardHistory(entries []clipboardEntry) {
	path, err := launtuiDataPath(clipboardHistoryFile)

	if err != nil {
		return
	}

	_ = saveJSON(path, entries)
}

type clipboardSuppression struct {
	Salt    string `json:"salt"`
	Hash    string `json:"hash"`
	Expires int64  `json:"expires"`
}

const suppressionWindow = 5 * time.Minute

func suppressClipboardRecording(text string) error {
	salt := make([]byte, 16)

	_, err := rand.Read(salt)

	if err != nil {
		return err
	}

	encodedSalt := hex.EncodeToString(salt)

	suppression := clipboardSuppression{
		Salt:    encodedSalt,
		Hash:    saltedHash(encodedSalt, text),
		Expires: time.Now().Add(suppressionWindow).Unix(),
	}

	path, err := launtuiCachePath(suppressionFile)

	if err != nil {
		return err
	}

	return saveJSON(path, suppression)
}

func saltedHash(salt, text string) string {
	digest := sha256.Sum256([]byte(salt + text))

	return hex.EncodeToString(digest[:])
}

func launtuiDataPath(name string) (string, error) {
	base := os.Getenv("XDG_DATA_HOME")

	if base == "" {
		home, err := os.UserHomeDir()

		if err != nil {
			return "", err
		}

		base = filepath.Join(home, ".local", "share")
	}

	return ensureDir(filepath.Join(base, "launtui"), name)
}

func launtuiCachePath(name string) (string, error) {
	base, err := os.UserCacheDir()

	if err != nil {
		return "", err
	}

	return ensureDir(filepath.Join(base, "launtui"), name)
}

func ensureDir(dir, name string) (string, error) {
	err := os.MkdirAll(dir, 0o700)

	if err != nil {
		return "", err
	}

	return filepath.Join(dir, name), nil
}

func loadJSON[T any](path string) (T, bool) {
	var value T

	data, err := os.ReadFile(path)

	if err != nil {
		return value, false
	}

	if json.Unmarshal(data, &value) != nil {
		return value, false
	}

	return value, true
}

func saveJSON(path string, value any) error {
	data, err := json.Marshal(value)

	if err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")

	if err != nil {
		return err
	}

	_, writeErr := temporary.Write(data)

	if closeErr := temporary.Close(); writeErr == nil {
		writeErr = closeErr
	}

	if writeErr != nil {
		_ = os.Remove(temporary.Name())

		return writeErr
	}

	return os.Rename(temporary.Name(), path)
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()

		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}

	return path
}

func collapseHome(path string) string {
	home, err := os.UserHomeDir()

	if err != nil || home == "" {
		return path
	}

	if path == home {
		return "~"
	}

	if rest := strings.TrimPrefix(path, home+string(filepath.Separator)); rest != path {
		return "~/" + rest
	}

	return path
}

func spawnDetached(dir string, argv ...string) {
	if len(argv) == 0 || argv[0] == "" {
		return
	}

	command := exec.Command(argv[0], argv[1:]...)
	command.Dir = dir
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	_ = command.Start()
}

func resolveTerminal(configured string) string {
	if configured != "" {
		return configured
	}

	if env := os.Getenv("TERMINAL"); env != "" {
		return env
	}

	for _, candidate := range []string{
		"foot", "alacritty", "kitty", "ghostty", "wezterm",
		"gnome-terminal", "konsole", "xfce4-terminal", "xterm",
	} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func terminalArgv(terminal, commandLine string) []string {
	if terminal == "" {
		return []string{"sh", "-c", commandLine}
	}

	switch filepath.Base(terminal) {
	case "foot", "kitty":
		return []string{terminal, "sh", "-c", commandLine}
	case "wezterm":
		return []string{terminal, "start", "--", "sh", "-c", commandLine}
	case "gnome-terminal":
		return []string{terminal, "--", "sh", "-c", commandLine}
	case "xfce4-terminal":
		return []string{terminal, "-x", "sh", "-c", commandLine}
	default:
		return []string{terminal, "-e", "sh", "-c", commandLine}
	}
}
