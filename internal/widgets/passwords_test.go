package widgets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPasswordsUsernameCache(t *testing.T) {
	mode, _ := NewPasswords(DefaultPasswordsConfig()).Update(passwordEntriesMsg{"github", "aws"})

	selected, cmd := mode.(Passwords).Select(0)

	if cmd == nil {
		t.Fatal("selecting an unresolved entry should return a decrypt command")
	}

	if right := selected.Rows()[0].right; right != "" {
		t.Fatalf("no username should show before it is resolved, got %q", right)
	}

	resolved, _ := selected.Update(passwordUsernameMsg{entry: "github", username: "me@example.com"})

	rows := resolved.Rows()

	if !strings.Contains(rows[0].right, "me@example.com") {
		t.Fatalf("the selected entry's resolved username should be shown, got %q", rows[0].right)
	}

	if rows[1].right != "" {
		t.Fatalf("a non-selected entry should show no username, got %q", rows[1].right)
	}

	passwords := resolved.(Passwords)

	if _, cmd := passwords.Select(0); cmd != nil {
		t.Fatal("selecting an already-resolved entry should not decrypt again")
	}

	movedAway, cmd := passwords.Select(1)

	if cmd == nil {
		t.Fatal("selecting an unresolved entry should return a decrypt command")
	}

	if right := movedAway.Rows()[0].right; right != "" {
		t.Fatalf("scrolling away should clear the previous row's username, got %q", right)
	}

	if _, cmd := passwords.Select(99); cmd != nil {
		t.Fatal("selecting out of range should be a no-op")
	}
}

func TestRunPasteSequence(t *testing.T) {
	cases := []struct {
		name           string
		payload        string
		clipboardTypes string
		wantErr        bool
		wantStages     []string
	}{
		{
			name:    "username then password",
			payload: "me@example.com\nhunter2",
			wantStages: []string{
				"--foreground --sensitive --paste-once|me@example.com",
				"--foreground --sensitive|hunter2",
			},
		},
		{
			name:       "password only",
			payload:    "\nhunter2",
			wantStages: []string{"--foreground --sensitive|hunter2"},
		},
		{
			name:           "aborts when the clipboard is taken over",
			payload:        "me@example.com\nhunter2",
			clipboardTypes: "text/plain",
			wantStages:     []string{"--foreground --sensitive --paste-once|me@example.com"},
		},
		{
			name:    "malformed payload",
			payload: "no newline",
			wantErr: true,
		},
	}

	systemPath := os.Getenv("PATH")

	for _, test := range cases {
		binDir := t.TempDir()
		logPath := filepath.Join(binDir, "stages.log")

		probesPath := filepath.Join(binDir, "probes.log")

		wlCopy := "#!/bin/sh\nprintf '%s|%s\\n' \"$*\" \"$(cat)\" >> \"" + logPath + "\"\n"

		wlPaste := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + probesPath + "\"\nexit 1\n"

		if test.clipboardTypes != "" {
			wlPaste = "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + probesPath + "\"\necho \"" + test.clipboardTypes + "\"\n"
		}

		if err := os.WriteFile(filepath.Join(binDir, "wl-copy"), []byte(wlCopy), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(binDir, "wl-paste"), []byte(wlPaste), 0o755); err != nil {
			t.Fatal(err)
		}

		t.Setenv("PATH", binDir+string(os.PathListSeparator)+systemPath)

		read, write, err := os.Pipe()

		if err != nil {
			t.Fatal(err)
		}

		if _, err := write.WriteString(test.payload); err != nil {
			t.Fatal(err)
		}

		write.Close()

		previousStdin := os.Stdin
		os.Stdin = read

		runErr := RunPasteSequence()

		os.Stdin = previousStdin
		read.Close()

		if (runErr != nil) != test.wantErr {
			t.Errorf("%s: RunPasteSequence() error = %v, wantErr %v", test.name, runErr, test.wantErr)

			continue
		}

		logged, _ := os.ReadFile(logPath)

		var stages []string

		if len(logged) > 0 {
			stages = strings.Split(strings.TrimRight(string(logged), "\n"), "\n")
		}

		if strings.Join(stages, ";") != strings.Join(test.wantStages, ";") {
			t.Errorf("%s: served stages = %q, want %q", test.name, stages, test.wantStages)
		}

		probes, _ := os.ReadFile(probesPath)

		for _, probe := range strings.Split(strings.TrimRight(string(probes), "\n"), "\n") {
			if probe != "" && probe != "--list-types" {
				t.Errorf("%s: wl-paste invoked with %q, only the non-consuming --list-types probe is allowed", test.name, probe)
			}
		}
	}
}
