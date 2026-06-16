package widgets

import (
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
