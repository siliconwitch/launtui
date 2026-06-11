//go:build darwin

package widgets

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const macAppScanDepth = 4

func launchArgv(app runApp, preferredTerminal string) []string {
	if app.Exec == "" {
		return nil
	}

	return []string{"open", app.Exec}
}

func scanInstalledApps() []runApp {
	seen := map[string]bool{}

	var apps []runApp

	for _, dir := range macAppDirs() {
		collectMacApps(dir, 0, seen, &apps)
	}

	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps
}

func macAppDirs() []string {
	dirs := []string{"/Applications", "/System/Applications"}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Applications"))
	}

	return dirs
}

func collectMacApps(dir string, depth int, seen map[string]bool, apps *[]runApp) {
	if depth > macAppScanDepth {
		return
	}

	entries, err := os.ReadDir(dir)

	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		if strings.HasSuffix(name, ".app") {
			appName := strings.TrimSuffix(name, ".app")
			key := strings.ToLower(appName)

			if seen[key] {
				continue
			}

			seen[key] = true
			*apps = append(*apps, runApp{Name: appName, Exec: path})

			continue
		}

		if entry.IsDir() && !strings.HasPrefix(name, ".") {
			collectMacApps(path, depth+1, seen, apps)
		}
	}
}
