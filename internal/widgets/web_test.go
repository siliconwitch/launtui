package widgets

import "testing"

func TestWebHistorySelection(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewWeb(DefaultWebConfig()).Update(webHistoryMsg{
		{Label: "Open https://github.com", URL: "https://github.com"},
		{Label: "Search the web for “go”", URL: "https://duckduckgo.com/?q=go"},
	})

	web := mode.SetQuery("google.com").(Web)

	activatedURL := func(index int) string {
		t.Helper()

		cmd := web.Activate(index)

		if cmd == nil {
			t.Fatalf("activating index %d should produce a command", index)
		}

		cmd()

		path, err := launtuiDataPath(webHistoryFile)

		if err != nil {
			t.Fatal(err)
		}

		saved, _ := loadJSON[[]webVisit](path)

		if len(saved) == 0 {
			t.Fatal("activation should record a visit")
		}

		return saved[0].URL
	}

	if got := activatedURL(0); got != "https://google.com" {
		t.Fatalf("live visit URL = %q, want the open action", got)
	}

	if got := activatedURL(2); got != "https://github.com" {
		t.Fatalf("first history visit URL = %q, want github", got)
	}

	if cmd := web.Activate(4); cmd != nil {
		t.Fatal("activating past the last row should do nothing")
	}
}

func TestWebRecall(t *testing.T) {
	mode, _ := NewWeb(DefaultWebConfig()).Update(webHistoryMsg{
		{Label: "Search the web for “go”", URL: "https://duckduckgo.com/?q=go", Query: "go"},
		{Label: "Open https://github.com", URL: "https://github.com"},
	})

	web := mode.SetQuery("google.com").(Web)

	cases := []struct {
		index int
		text  string
		ok    bool
	}{
		{0, "", false},
		{1, "", false},
		{2, "go", true},
		{3, "https://github.com", true},
		{4, "", false},
	}

	for _, testCase := range cases {
		text, ok := web.RecallText(testCase.index)

		if text != testCase.text || ok != testCase.ok {
			t.Errorf("RecallText(%d) = (%q, %v), want (%q, %v)", testCase.index, text, ok, testCase.text, testCase.ok)
		}
	}
}

func TestWebDeleteSelectedHistory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewWeb(DefaultWebConfig()).Update(webHistoryMsg{
		{Label: "first", URL: "https://a.example"},
		{Label: "second", URL: "https://b.example"},
	})

	deleted, cmd := mode.(Web).DeleteRow(0)

	if cmd == nil {
		t.Fatal("deleting a history entry should be persisted")
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

	if _, cmd := typed.DeleteRow(0); cmd != nil {
		t.Fatal("deleting a live action should do nothing")
	}
}

func TestWebClearHistory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewWeb(DefaultWebConfig()).Update(webHistoryMsg{
		{Label: "first", URL: "https://a.example"},
		{Label: "second", URL: "https://b.example"},
	})

	cleared, cmd := mode.(Web).ClearRows()

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

func TestWebRecordsVisitsAndDeduplicates(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	web := NewWeb(DefaultWebConfig())

	activate := func(query string) {
		t.Helper()

		cmd := web.SetQuery(query).(Web).Activate(0)

		if cmd == nil {
			t.Fatalf("activating %q should produce a command", query)
		}

		cmd()
	}

	activate("a.example")
	activate("b.example")
	activate("a.example")

	path, err := launtuiDataPath(webHistoryFile)

	if err != nil {
		t.Fatal(err)
	}

	saved, _ := loadJSON[[]webVisit](path)

	if len(saved) != 2 || saved[0].URL != "https://a.example" || saved[1].URL != "https://b.example" {
		t.Fatalf("saved history = %+v, want a.example moved to front with b.example deduplicated", saved)
	}

	if saved[0].Query != "a.example" {
		t.Fatalf("saved query = %q, want the typed query recorded for recall", saved[0].Query)
	}
}

func TestWebActions(t *testing.T) {
	web := NewWeb(DefaultWebConfig())

	empty := web.SetQuery("").(Web)

	if HasResults(empty.Rows()) {
		t.Fatal("empty query should produce no actions")
	}

	question := web.SetQuery("how do I update go").(Web)

	if len(question.actions) != 1 {
		t.Fatalf("question actions = %d, want search only", len(question.actions))
	}

	if question.actions[0].url != "https://duckduckgo.com/?q=how+do+I+update+go" {
		t.Fatalf("search url = %q", question.actions[0].url)
	}

	urls := map[string]string{
		"google.com":            "https://google.com",
		"google.com/search?q=x": "https://google.com/search?q=x",
		"http://example.org":    "http://example.org",
		"https://example.org/a": "https://example.org/a",
		"localhost":             "http://localhost",
		"localhost:3000":        "http://localhost:3000",
		"sub.domain.co.uk:8080": "https://sub.domain.co.uk:8080",
	}

	for input, want := range urls {
		actions := web.SetQuery(input).(Web).actions

		if len(actions) != 2 {
			t.Fatalf("SetQuery(%q) actions = %d, want open + search", input, len(actions))
		}

		if actions[0].url != want {
			t.Errorf("SetQuery(%q) open url = %q, want %q", input, actions[0].url, want)
		}
	}

	notURLs := []string{"how can I update my go version", "hello", "btop", "1.5", "a.b", "localhost3000"}

	for _, input := range notURLs {
		if actions := web.SetQuery(input).(Web).actions; len(actions) != 1 {
			t.Errorf("SetQuery(%q) actions = %d, want search only", input, len(actions))
		}
	}
}
