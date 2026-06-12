package widgets

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	clipboardHistoryFile  = "clipboard-history.json"
	suppressionFile       = "suppressed.json"
	defaultClipboardLimit = 100
)

type clipboardEntry struct {
	Text string `json:"text"`
	Time int64  `json:"time"`
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

		cmd := exec.Command(path, tool[1:]...)
		cmd.Stdin = strings.NewReader(text)

		if cmd.Run() == nil {
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

func loadClipboardHistory() []clipboardEntry {
	path, err := launtuiDataPath(clipboardHistoryFile)

	if err != nil {
		return nil
	}

	entries, _ := loadJSON[[]clipboardEntry](path)

	return entries
}

func recordClipboardText(text string, limit int) []clipboardEntry {
	if strings.TrimSpace(text) == "" || clipboardRecordingSuppressed(text) {
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

func clipboardRecordingSuppressed(text string) bool {
	path, err := launtuiCachePath(suppressionFile)

	if err != nil {
		return false
	}

	suppression, ok := loadJSON[clipboardSuppression](path)

	if !ok {
		return false
	}

	if time.Now().Unix() > suppression.Expires {
		_ = os.Remove(path)

		return false
	}

	return saltedHash(suppression.Salt, text) == suppression.Hash
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

func spawnDetached(dir string, argv ...string) {
	if len(argv) == 0 || argv[0] == "" {
		return
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	_ = cmd.Start()
}
