//go:build darwin

package widgets

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func safariSupported() bool { return true }

func loadSafariEntries(cfg SafariConfig) ([]safariEntry, bool) {
	seen := map[string]bool{}

	var entries []safariEntry

	add := func(found []safariEntry) {
		for _, entry := range found {
			if entry.url == "" || seen[entry.url] {
				continue
			}

			seen[entry.url] = true
			entries = append(entries, entry)
		}
	}

	add(loadSafariTabs())

	bookmarks, bookmarksDenied := loadSafariBookmarks()
	add(bookmarks)

	history, historyDenied := loadSafariHistory(cfg.HistoryLimit)
	add(history)

	return entries, bookmarksDenied || historyDenied
}

func activateSafariEntry(entry safariEntry) {
	if entry.kind == safariTab {
		focusSafariTab(entry.window, entry.tab)

		return
	}

	openURL(entry.url)
}

const safariTabsScript = `set output to ""
if application "Safari" is running then
	tell application "Safari"
		repeat with w from 1 to (count of windows)
			repeat with t from 1 to (count of tabs of window w)
				try
					set theTab to tab t of window w
					set output to output & w & tab & t & tab & (URL of theTab) & tab & (name of theTab) & linefeed
				end try
			end repeat
		end repeat
	end tell
end if
return output`

func loadSafariTabs() []safariEntry {
	output, err := exec.Command("osascript", "-e", safariTabsScript).Output()

	if err != nil {
		return nil
	}

	return parseSafariTabs(string(output))
}

func parseSafariTabs(output string) []safariEntry {
	var entries []safariEntry

	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.SplitN(line, "\t", 4)

		if len(fields) < 4 {
			continue
		}

		window, windowErr := strconv.Atoi(strings.TrimSpace(fields[0]))
		tab, tabErr := strconv.Atoi(strings.TrimSpace(fields[1]))

		if windowErr != nil || tabErr != nil {
			continue
		}

		url := strings.TrimSpace(fields[2])
		title := strings.TrimSpace(fields[3])

		if url == "" {
			continue
		}

		if title == "" {
			title = url
		}

		entries = append(entries, safariEntry{kind: safariTab, title: title, url: url, window: window, tab: tab})
	}

	return entries
}

func focusSafariTab(window, tab int) {
	w := strconv.Itoa(window)
	t := strconv.Itoa(tab)

	script := "tell application \"Safari\"\n" +
		"set current tab of window " + w + " to tab " + t + " of window " + w + "\n" +
		"set index of window " + w + " to 1\n" +
		"activate\n" +
		"end tell"

	_ = exec.Command("osascript", "-e", script).Run()
}

type safariBookmarkNode struct {
	WebBookmarkType string `json:"WebBookmarkType"`
	URLString       string `json:"URLString"`
	URIDictionary   struct {
		Title string `json:"title"`
	} `json:"URIDictionary"`
	Children []safariBookmarkNode `json:"Children"`
}

func loadSafariBookmarks() ([]safariEntry, bool) {
	home, err := os.UserHomeDir()

	if err != nil {
		return nil, false
	}

	path := filepath.Join(home, "Library", "Safari", "Bookmarks.plist")

	output, err := exec.Command("plutil", "-convert", "json", "-o", "-", path).Output()

	if err != nil {
		return nil, true
	}

	return parseSafariBookmarks(output), false
}

func parseSafariBookmarks(data []byte) []safariEntry {
	var root safariBookmarkNode

	if json.Unmarshal(data, &root) != nil {
		return nil
	}

	var entries []safariEntry

	var walk func(node safariBookmarkNode)

	walk = func(node safariBookmarkNode) {
		if node.WebBookmarkType == "WebBookmarkTypeLeaf" && node.URLString != "" {
			title := node.URIDictionary.Title

			if title == "" {
				title = node.URLString
			}

			entries = append(entries, safariEntry{kind: safariBookmark, title: title, url: node.URLString})
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return entries
}

func loadSafariHistory(limit int) ([]safariEntry, bool) {
	if limit <= 0 {
		limit = 2000
	}

	home, err := os.UserHomeDir()

	if err != nil {
		return nil, false
	}

	source := filepath.Join(home, "Library", "Safari", "History.db")

	copyPath, err := copySafariHistory(source)

	if err != nil {
		return nil, true
	}

	defer os.RemoveAll(filepath.Dir(copyPath))

	query := "SELECT i.url, IFNULL(v.title, '') FROM history_items i " +
		"JOIN history_visits v ON v.history_item = i.id " +
		"GROUP BY i.id ORDER BY MAX(v.visit_time) DESC LIMIT " + strconv.Itoa(limit) + ";"

	output, err := exec.Command("sqlite3", "-csv", copyPath, query).Output()

	if err != nil {
		return nil, false
	}

	return parseSafariHistoryCSV(output), false
}

func copySafariHistory(source string) (string, error) {
	dir, err := os.MkdirTemp("", "launtui-safari")

	if err != nil {
		return "", err
	}

	dest := filepath.Join(dir, "History.db")

	for _, suffix := range []string{"", "-wal", "-shm"} {
		data, err := os.ReadFile(source + suffix)

		if err != nil {
			if suffix == "" {
				os.RemoveAll(dir)

				return "", err
			}

			continue
		}

		if err := os.WriteFile(dest+suffix, data, 0o600); err != nil {
			os.RemoveAll(dir)

			return "", err
		}
	}

	return dest, nil
}

func parseSafariHistoryCSV(data []byte) []safariEntry {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()

	if err != nil {
		return nil
	}

	var entries []safariEntry

	for _, record := range records {
		if len(record) == 0 || record[0] == "" {
			continue
		}

		title := ""

		if len(record) > 1 {
			title = record[1]
		}

		if title == "" {
			title = record[0]
		}

		entries = append(entries, safariEntry{kind: safariHistory, title: title, url: record[0]})
	}

	return entries
}
