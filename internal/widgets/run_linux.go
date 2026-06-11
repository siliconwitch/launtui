//go:build !darwin

package widgets

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func launchArgv(app runApp, preferredTerminal string) []string {
	cmdline := strings.TrimSpace(app.Exec)

	if cmdline == "" {
		return nil
	}

	if app.Terminal {
		return terminalArgv(resolveTerminal(preferredTerminal), cmdline)
	}

	return []string{"sh", "-c", cmdline}
}

func resolveTerminal(preferred string) string {
	if preferred != "" {
		return preferred
	}

	if terminal := os.Getenv("TERMINAL"); terminal != "" {
		return terminal
	}

	candidates := []string{
		"foot", "alacritty", "kitty", "ghostty", "wezterm",
		"gnome-terminal", "konsole", "xfce4-terminal", "xterm",
	}

	for _, candidate := range candidates {
		_, err := exec.LookPath(candidate)

		if err == nil {
			return candidate
		}
	}

	return ""
}

func terminalArgv(terminal, cmdline string) []string {
	if terminal == "" {
		return []string{"sh", "-c", cmdline}
	}

	switch filepath.Base(terminal) {
	case "foot", "kitty":
		return []string{terminal, "sh", "-c", cmdline}
	case "wezterm":
		return []string{terminal, "start", "--", "sh", "-c", cmdline}
	case "gnome-terminal":
		return []string{terminal, "--", "sh", "-c", cmdline}
	case "xfce4-terminal":
		return []string{terminal, "-x", "sh", "-c", cmdline}
	default:
		return []string{terminal, "-e", "sh", "-c", cmdline}
	}
}

func applicationDirs() []string {
	var dirs []string

	dataHome := os.Getenv("XDG_DATA_HOME")

	if dataHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}

	if dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "applications"))
	}

	dataDirs := os.Getenv("XDG_DATA_DIRS")

	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}

	for _, dir := range filepath.SplitList(dataDirs) {
		if dir != "" {
			dirs = append(dirs, filepath.Join(dir, "applications"))
		}
	}

	return dirs
}

func scanInstalledApps() []runApp {
	seen := map[string]bool{}

	var apps []runApp

	for _, dir := range applicationDirs() {
		entries, err := os.ReadDir(dir)

		if err != nil {
			continue
		}

		for _, entry := range entries {
			id := entry.Name()

			if entry.IsDir() || !strings.HasSuffix(id, ".desktop") || seen[id] {
				continue
			}

			seen[id] = true

			if app, ok := parseDesktopFile(filepath.Join(dir, id)); ok {
				apps = append(apps, app)
			}
		}
	}

	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps
}

var execFieldCodes = regexp.MustCompile(`%[fFuUdDnNickvm]`)

func parseDesktopFile(path string) (runApp, bool) {
	file, err := os.Open(path)

	if err != nil {
		return runApp{}, false
	}

	defer file.Close()

	var (
		app        runApp
		entryType  string
		inEntry    bool
		noDisplay  bool
		isHidden   bool
		tryExec    string
		onlyShowIn string
		notShowIn  string
	)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}

		if !inEntry {
			continue
		}

		key, value, ok := strings.Cut(line, "=")

		if !ok {
			continue
		}

		switch strings.TrimSpace(key) {
		case "Type":
			entryType = strings.TrimSpace(value)
		case "Name":
			app.Name = strings.TrimSpace(value)
		case "Comment":
			app.Comment = strings.TrimSpace(value)
		case "Exec":
			app.Exec = stripFieldCodes(value)
		case "TryExec":
			tryExec = strings.TrimSpace(value)
		case "Path":
			app.WorkingDir = strings.TrimSpace(value)
		case "Terminal":
			app.Terminal = strings.TrimSpace(value) == "true"
		case "NoDisplay":
			noDisplay = strings.TrimSpace(value) == "true"
		case "Hidden":
			isHidden = strings.TrimSpace(value) == "true"
		case "OnlyShowIn":
			onlyShowIn = strings.TrimSpace(value)
		case "NotShowIn":
			notShowIn = strings.TrimSpace(value)
		}
	}

	if scanner.Err() != nil {
		return runApp{}, false
	}

	if entryType != "Application" || noDisplay || isHidden || app.Name == "" || app.Exec == "" {
		return runApp{}, false
	}

	if !desktopVisibleIn(onlyShowIn, notShowIn, os.Getenv("XDG_CURRENT_DESKTOP")) {
		return runApp{}, false
	}

	if tryExec != "" {
		_, err := exec.LookPath(tryExec)

		if err != nil {
			return runApp{}, false
		}
	}

	return app, true
}

func desktopVisibleIn(onlyShowIn, notShowIn, currentDesktop string) bool {
	desktops := map[string]bool{}

	for _, name := range strings.Split(currentDesktop, ":") {
		if name != "" {
			desktops[strings.ToLower(name)] = true
		}
	}

	if onlyShowIn != "" {
		for _, name := range strings.Split(onlyShowIn, ";") {
			if name != "" && desktops[strings.ToLower(name)] {
				return true
			}
		}

		return false
	}

	for _, name := range strings.Split(notShowIn, ";") {
		if name != "" && desktops[strings.ToLower(name)] {
			return false
		}
	}

	return true
}

func stripFieldCodes(execLine string) string {
	execLine = strings.ReplaceAll(execLine, "%%", "\x00")
	execLine = execFieldCodes.ReplaceAllString(execLine, "")
	execLine = strings.ReplaceAll(execLine, `""`, "")
	execLine = strings.ReplaceAll(execLine, `''`, "")
	execLine = strings.ReplaceAll(execLine, "\x00", "%")

	return strings.TrimSpace(execLine)
}
