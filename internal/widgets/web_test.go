package widgets

import "testing"

func TestQueryAsURL(t *testing.T) {
	valid := map[string]string{
		"google.com":            "https://google.com",
		"google.com/search?q=x": "https://google.com/search?q=x",
		"http://example.org":    "http://example.org",
		"https://example.org/a": "https://example.org/a",
		"localhost":             "http://localhost",
		"localhost:3000":        "http://localhost:3000",
		"sub.domain.co.uk:8080": "https://sub.domain.co.uk:8080",
	}

	for input, want := range valid {
		got, ok := queryAsURL(input)

		if !ok || got != want {
			t.Errorf("queryAsURL(%q) = %q (ok=%v), want %q", input, got, ok, want)
		}
	}

	invalid := []string{"how can I update my go version", "hello", "btop", "1.5", "a.b", "localhost3000"}

	for _, input := range invalid {
		if got, ok := queryAsURL(input); ok {
			t.Errorf("queryAsURL(%q) = %q, want no match", input, got)
		}
	}
}

func TestWebHistorySelection(t *testing.T) {
	mode, _ := NewWeb(DefaultWebConfig()).Update(webHistoryMsg{
		{Label: "Open https://github.com", URL: "https://github.com"},
		{Label: "Search the web for “go”", URL: "https://duckduckgo.com/?q=go"},
	})

	web := mode.SetQuery("google.com").(Web)

	if visit, ok := web.selectedVisit(); !ok || visit.URL != "https://google.com" {
		t.Fatalf("live visit = %+v (ok=%v), want the open action", visit, ok)
	}

	first := web.MoveDown().MoveDown().(Web)

	if visit, ok := first.selectedVisit(); !ok || visit.URL != "https://github.com" {
		t.Fatalf("first history visit = %+v (ok=%v)", visit, ok)
	}

	clamped := first.MoveDown().MoveDown().(Web)

	if clamped.cursor != 3 {
		t.Fatalf("cursor = %d, should clamp at the last history entry", clamped.cursor)
	}
}

func TestWebDeleteSelectedHistory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewWeb(DefaultWebConfig()).Update(webHistoryMsg{
		{Label: "first", URL: "https://a.example"},
		{Label: "second", URL: "https://b.example"},
	})

	deleted, cmd, handled := mode.(Web).DeleteSelectedHistory()

	if !handled || cmd == nil {
		t.Fatal("deleting a history entry should be handled and persisted")
	}

	web := deleted.(Web)

	if len(web.history) != 1 || web.history[0].URL != "https://b.example" {
		t.Fatalf("history after delete = %+v", web.history)
	}

	cmd()

	path, err := launtuiDataPath(webHistoryFile)

	if err != nil {
		t.Fatal(err)
	}

	saved, _ := loadJSON[[]webVisit](path)

	if len(saved) != 1 || saved[0].URL != "https://b.example" {
		t.Fatalf("saved history = %+v", saved)
	}

	typed := web.SetQuery("google.com").(Web)

	if _, _, handled := typed.DeleteSelectedHistory(); handled {
		t.Fatal("delete on a live action should not be handled")
	}
}

func TestWebClearHistory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewWeb(DefaultWebConfig()).Update(webHistoryMsg{
		{Label: "first", URL: "https://a.example"},
		{Label: "second", URL: "https://b.example"},
	})

	cleared, cmd := mode.(Web).ClearHistory()

	if len(cleared.(Web).history) != 0 {
		t.Fatalf("history after clear = %+v", cleared.(Web).history)
	}

	cmd()

	path, err := launtuiDataPath(webHistoryFile)

	if err != nil {
		t.Fatal(err)
	}

	if saved, _ := loadJSON[[]webVisit](path); len(saved) != 0 {
		t.Fatalf("saved history after clear = %+v", saved)
	}
}

func TestRecordWebVisitDeduplicatesByURL(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	recordWebVisit(webVisit{Label: "first", URL: "https://a.example", Time: 1}, 10)
	recordWebVisit(webVisit{Label: "second", URL: "https://b.example", Time: 2}, 10)
	recordWebVisit(webVisit{Label: "first", URL: "https://a.example", Time: 3}, 10)

	path, err := launtuiDataPath(webHistoryFile)

	if err != nil {
		t.Fatal(err)
	}

	saved, _ := loadJSON[[]webVisit](path)

	if len(saved) != 2 || saved[0].URL != "https://a.example" || saved[1].URL != "https://b.example" {
		t.Fatalf("saved history = %+v", saved)
	}
}

func TestWebActions(t *testing.T) {
	web := NewWeb(DefaultWebConfig())

	empty := web.SetQuery("").(Web)

	if empty.HasResults() {
		t.Fatal("empty query should produce no actions")
	}

	address := web.SetQuery("google.com").(Web)

	if len(address.actions) != 2 {
		t.Fatalf("address actions = %d, want open + search", len(address.actions))
	}

	if address.actions[0].url != "https://google.com" {
		t.Fatalf("open url = %q", address.actions[0].url)
	}

	question := web.SetQuery("how do I update go").(Web)

	if len(question.actions) != 1 {
		t.Fatalf("question actions = %d, want search only", len(question.actions))
	}

	if question.actions[0].url != "https://duckduckgo.com/?q=how+do+I+update+go" {
		t.Fatalf("search url = %q", question.actions[0].url)
	}
}
