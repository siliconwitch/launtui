//go:build darwin

package widgets

import "testing"

func TestParseSafariTabs(t *testing.T) {
	output := "1\t1\thttps://example.com/\tExample Domain\n" +
		"1\t2\thttps://go.dev/\tThe Go Programming Language\n" +
		"bad line without tabs\n" +
		"2\t1\thttps://news.example.com/\t\n"

	tabs := parseSafariTabs(output)

	if len(tabs) != 3 {
		t.Fatalf("tabs = %+v", tabs)
	}

	if tabs[0].window != 1 || tabs[0].tab != 1 || tabs[0].url != "https://example.com/" || tabs[0].title != "Example Domain" {
		t.Fatalf("tab 0 = %+v", tabs[0])
	}

	if tabs[2].title != "https://news.example.com/" {
		t.Fatalf("empty title should fall back to url, got %q", tabs[2].title)
	}

	for _, tab := range tabs {
		if tab.kind != safariTab {
			t.Fatalf("kind = %v, want tab", tab.kind)
		}
	}
}

func TestParseSafariBookmarks(t *testing.T) {
	data := []byte(`{
		"WebBookmarkType": "WebBookmarkTypeList",
		"Children": [
			{
				"WebBookmarkType": "WebBookmarkTypeList",
				"Title": "BookmarksBar",
				"Children": [
					{"WebBookmarkType":"WebBookmarkTypeLeaf","URLString":"https://example.com/","URIDictionary":{"title":"Example"}},
					{"WebBookmarkType":"WebBookmarkTypeLeaf","URLString":"https://no-title.example/","URIDictionary":{}}
				]
			}
		]
	}`)

	bookmarks := parseSafariBookmarks(data)

	if len(bookmarks) != 2 {
		t.Fatalf("bookmarks = %+v", bookmarks)
	}

	if bookmarks[0].title != "Example" || bookmarks[0].url != "https://example.com/" || bookmarks[0].kind != safariBookmark {
		t.Fatalf("bookmark 0 = %+v", bookmarks[0])
	}

	if bookmarks[1].title != "https://no-title.example/" {
		t.Fatalf("missing title should fall back to url, got %q", bookmarks[1].title)
	}
}

func TestParseSafariHistoryCSV(t *testing.T) {
	data := []byte("https://example.com/,Example Domain\n" +
		"https://commas.example/,\"Title, with comma\"\n" +
		"https://no-title.example/,\n")

	history := parseSafariHistoryCSV(data)

	if len(history) != 3 {
		t.Fatalf("history = %+v", history)
	}

	if history[1].title != "Title, with comma" {
		t.Fatalf("csv quoting not handled, got %q", history[1].title)
	}

	if history[2].title != "https://no-title.example/" {
		t.Fatalf("empty title should fall back to url, got %q", history[2].title)
	}

	if history[0].kind != safariHistory {
		t.Fatalf("kind = %v, want history", history[0].kind)
	}
}
