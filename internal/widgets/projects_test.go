package widgets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectsLoadsDirsAndProjects(t *testing.T) {
	root := t.TempDir()

	mkdir := func(parts ...string) string {
		t.Helper()

		path := filepath.Join(append([]string{root}, parts...)...)

		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}

		return path
	}

	mkdir("projects", "alpha", ".git")
	mkdir("projects", "beta")
	mkdir("work", "alpha")
	mkdir("company")

	config := DefaultProjectsConfig()
	config.Dirs = []string{filepath.Join(root, "projects"), filepath.Join(root, "work")}
	config.Projects = []string{filepath.Join(root, "company")}

	loaded, ok := NewProjects(config).Init()().(projectsLoadedMsg)

	if !ok {
		t.Fatal("Init should return a projectsLoadedMsg")
	}

	if len(loaded) != 4 {
		t.Fatalf("loaded %d projects, want 4: %+v", len(loaded), loaded)
	}

	byPath := map[string]project{}

	for _, proj := range loaded {
		byPath[proj.path] = proj
	}

	if !byPath[filepath.Join(root, "projects", "alpha")].git {
		t.Error("projects/alpha should be detected as a git repo")
	}

	if byPath[filepath.Join(root, "projects", "beta")].git {
		t.Error("projects/beta should not be a git repo")
	}

	if label := byPath[filepath.Join(root, "company")].label; label != "company" {
		t.Errorf("unique project label = %q, want company", label)
	}

	projectsAlpha := byPath[filepath.Join(root, "projects", "alpha")]
	workAlpha := byPath[filepath.Join(root, "work", "alpha")]

	if projectsAlpha.label == "alpha" || workAlpha.label == "alpha" {
		t.Errorf("colliding names should show the full path, got %q and %q", projectsAlpha.label, workAlpha.label)
	}

	if !strings.Contains(projectsAlpha.label, "projects") || !strings.Contains(workAlpha.label, "work") {
		t.Errorf("collision labels should disambiguate by path: %q vs %q", projectsAlpha.label, workAlpha.label)
	}
}

func TestProjectsActivateWithoutEditor(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	mode, _ := NewProjects(DefaultProjectsConfig()).Update(projectsLoadedMsg{
		{label: "x", path: "/tmp/x"},
	})

	cmd := mode.Activate(0)

	if cmd == nil {
		t.Fatal("activating should return a command")
	}

	if _, ok := cmd().(editorMissingMsg); !ok {
		t.Fatal("activating with no editor should report a missing editor")
	}
}

func TestParseGitStatus(t *testing.T) {
	output := "# branch.oid abc123\n" +
		"# branch.head main\n" +
		"# branch.upstream origin/main\n" +
		"# branch.ab +2 -1\n" +
		"1 .M N... 100644 100644 100644 abc def internal/file.go\n"

	status := parseGitStatus(output)

	if status.branch != "main" || !status.dirty || status.ahead != 2 || status.behind != 1 {
		t.Fatalf("status = %+v", status)
	}

	clean := parseGitStatus("# branch.oid abc\n# branch.head trunk\n# branch.ab +0 -0\n")

	if clean.branch != "trunk" || clean.dirty || clean.ahead != 0 || clean.behind != 0 {
		t.Fatalf("clean status = %+v", clean)
	}

	detached := parseGitStatus("# branch.oid abc\n# branch.head (detached)\n")

	if detached.branch != "(detached)" {
		t.Fatalf("detached status = %+v", detached)
	}
}
