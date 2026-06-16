package widgets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunExcludesApps(t *testing.T) {
	config := DefaultRunConfig()
	config.Exclude = []string{"firefox", "  Slack  "}

	run := NewRun(config)

	updated, _ := run.Update(appsLoadedMsg{
		{Name: "Firefox", Exec: "firefox"},
		{Name: "Slack", Exec: "slack"},
		{Name: "Terminal", Exec: "term"},
	})

	apps := updated.(Run).list.items

	if len(apps) != 1 || apps[0].Name != "Terminal" {
		t.Fatalf("apps = %+v, want only Terminal", apps)
	}
}

func TestRunCommentToggle(t *testing.T) {
	apps := appsLoadedMsg{{Name: "Editor", Comment: "Edit text", Exec: "ed"}}

	shown, _ := NewRun(DefaultRunConfig()).Update(apps)

	if shown.(Run).Rows()[0].right == "" {
		t.Fatal("comment should be shown by default")
	}

	config := DefaultRunConfig()
	config.Comment = false

	hidden, _ := NewRun(config).Update(apps)

	if right := hidden.(Run).Rows()[0].right; right != "" {
		t.Fatalf("comment should be hidden when disabled, got %q", right)
	}
}

func TestParseDesktopFileStripsExecFieldCodes(t *testing.T) {
	cases := map[string]string{
		"firefox %u":                                  "firefox",
		"code --new-window %F":                        "code --new-window",
		"app %%foo":                                   "app %foo",
		"/usr/bin/foo -a %i %c":                       "/usr/bin/foo -a",
		`vlc --started-from-file "%U"`:                "vlc --started-from-file",
		"mpv --player-operation-mode=pseudo-gui '%U'": "mpv --player-operation-mode=pseudo-gui",
	}

	dir := t.TempDir()

	for exec, want := range cases {
		path := filepath.Join(dir, "entry.desktop")
		contents := "[Desktop Entry]\nType=Application\nName=Test\nExec=" + exec + "\n"

		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}

		app, ok := parseDesktopFile(path)

		if !ok || app.Exec != want {
			t.Errorf("parseDesktopFile(Exec=%q) = %q (ok=%v), want %q", exec, app.Exec, ok, want)
		}
	}
}

func TestDesktopVisibleIn(t *testing.T) {
	if desktopVisibleIn("GNOME;KDE;", "", "niri") {
		t.Error("OnlyShowIn=GNOME;KDE should hide the entry on niri")
	}

	if !desktopVisibleIn("GNOME;KDE;", "", "KDE") {
		t.Error("OnlyShowIn=GNOME;KDE should show the entry on KDE")
	}

	if desktopVisibleIn("", "GNOME;", "GNOME") {
		t.Error("NotShowIn=GNOME should hide the entry on GNOME")
	}

	if !desktopVisibleIn("", "GNOME;", "niri") {
		t.Error("NotShowIn=GNOME should show the entry on niri")
	}

	if !desktopVisibleIn("", "", "") {
		t.Error("entries without ShowIn keys should always be visible")
	}
}

func TestParseDesktopFile(t *testing.T) {
	dir := t.TempDir()

	write := func(name, body string) string {
		t.Helper()

		path := filepath.Join(dir, name)

		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}

		return path
	}

	good := write("good.desktop",
		"[Desktop Entry]\nType=Application\nName=Cool App\nComment=Does cool things\nExec=coolapp %U\n")

	app, ok := parseDesktopFile(good)

	if !ok {
		t.Fatal("good.desktop should parse")
	}

	if app.Name != "Cool App" || app.Exec != "coolapp" || app.Comment != "Does cool things" {
		t.Fatalf("parsed = %+v", app)
	}

	if app.Terminal {
		t.Fatal("good.desktop should not be a terminal app")
	}

	terminal := write("terminal.desktop",
		"[Desktop Entry]\nType=Application\nName=btop++\nExec=btop\nTerminal=true\nPath=/tmp\n")

	terminalApp, ok := parseDesktopFile(terminal)

	if !ok {
		t.Fatal("terminal.desktop should parse")
	}

	if !terminalApp.Terminal || terminalApp.WorkingDir != "/tmp" {
		t.Fatalf("parsed terminal app = %+v", terminalApp)
	}

	skipped := map[string]string{
		"hidden.desktop":     "[Desktop Entry]\nType=Application\nName=N\nExec=n\nNoDisplay=true\n",
		"link.desktop":       "[Desktop Entry]\nType=Link\nName=N\nURL=http://x\n",
		"noexec.desktop":     "[Desktop Entry]\nType=Application\nName=N\n",
		"truehidden.desktop": "[Desktop Entry]\nType=Application\nName=N\nExec=n\nHidden=true\n",
		"missing.desktop":    "[Desktop Entry]\nType=Application\nName=N\nExec=n\nTryExec=launtui-no-such-binary\n",
	}

	for name, body := range skipped {
		if _, ok := parseDesktopFile(write(name, body)); ok {
			t.Errorf("%s should be skipped", name)
		}
	}
}
