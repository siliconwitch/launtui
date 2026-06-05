package widgets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripFieldCodes(t *testing.T) {
	cases := map[string]string{
		"firefox %u":            "firefox",
		"code --new-window %F":  "code --new-window",
		"app %%foo":             "app %foo",
		"/usr/bin/foo -a %i %c": "/usr/bin/foo -a",
	}

	for in, want := range cases {
		if got := stripFieldCodes(in); got != want {
			t.Errorf("stripFieldCodes(%q) = %q, want %q", in, got, want)
		}
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

	skipped := map[string]string{
		"hidden.desktop":     "[Desktop Entry]\nType=Application\nName=N\nExec=n\nNoDisplay=true\n",
		"link.desktop":       "[Desktop Entry]\nType=Link\nName=N\nURL=http://x\n",
		"noexec.desktop":     "[Desktop Entry]\nType=Application\nName=N\n",
		"truehidden.desktop": "[Desktop Entry]\nType=Application\nName=N\nExec=n\nHidden=true\n",
	}

	for name, body := range skipped {
		if _, ok := parseDesktopFile(write(name, body)); ok {
			t.Errorf("%s should be skipped", name)
		}
	}
}
