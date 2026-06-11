package widgets

import "testing"

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

func TestEditorArgv(t *testing.T) {
	cases := map[string][]string{
		"hx":            {"hx", "."},
		"/usr/bin/nvim": {"/usr/bin/nvim", "."},
		"code --wait":   {"code", "--wait", "."},
		"nano":          {"nano"},
		"micro":         {"micro"},
	}

	for editor, want := range cases {
		got := editorArgv(editor)

		if len(got) != len(want) {
			t.Errorf("editorArgv(%q) = %v, want %v", editor, got, want)
			continue
		}

		for i := range want {
			if got[i] != want[i] {
				t.Errorf("editorArgv(%q) = %v, want %v", editor, got, want)
				break
			}
		}
	}
}

func TestProjectsCursorResetsOnQueryChange(t *testing.T) {
	mode, _ := NewProjects(DefaultProjectsConfig()).Update(projectsLoadedMsg{
		{name: "alpha"},
		{name: "beta"},
		{name: "gamma"},
	})

	moved := mode.MoveDown().MoveDown().(Projects)

	if moved.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", moved.cursor)
	}

	typed := moved.SetQuery("a").(Projects)

	if typed.cursor != 0 {
		t.Fatalf("cursor after typing = %d, want 0", typed.cursor)
	}
}
