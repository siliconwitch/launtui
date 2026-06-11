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
