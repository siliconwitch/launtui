package widgets

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestEmojiCatalogue(t *testing.T) {
	var catalogue struct {
		License string       `json:"license"`
		Emoji   []emojiEntry `json:"emoji"`
	}

	err := json.Unmarshal(emojiCatalogue, &catalogue)

	if err != nil {
		t.Fatalf("embedded catalogue does not parse: %v", err)
	}

	if catalogue.License == "" {
		t.Fatal("embedded catalogue has no license")
	}

	if len(catalogue.Emoji) < 1000 {
		t.Fatalf("embedded catalogue has %d emoji, want at least 1000", len(catalogue.Emoji))
	}
}

func TestEmojiSearch(t *testing.T) {
	emoji := NewEmoji(DefaultEmojiConfig())

	tests := []struct {
		query     string
		wantFirst string
	}{
		{query: "eyes", wantFirst: "👀"},
		{query: "cat", wantFirst: "🐈"},
		{query: "food", wantFirst: "🍇"},
		{query: "fruit", wantFirst: "🥝"},
		{query: "closed", wantFirst: "🌂"},
		{query: "hat", wantFirst: "🎩"},
		{query: "o'clock", wantFirst: "🕛"},
		{query: "curacao", wantFirst: "🇨🇼"},
		{query: "Curaçao", wantFirst: "🇨🇼"},
		{query: "  RED   heart ", wantFirst: "❤️"},
		{query: "up thumbs", wantFirst: "👍"},
		{query: "!!!", wantFirst: ""},
		{query: "zzqx", wantFirst: ""},
	}

	for _, test := range tests {
		results := emoji.SetQuery(test.query).(Emoji).results

		first := ""

		if len(results) > 0 {
			first = results[0].Glyph
		}

		if first != test.wantFirst {
			t.Errorf("query %q: first result = %q, want %q", test.query, first, test.wantFirst)
		}
	}
}

func TestEmojiRecent(t *testing.T) {
	mode, _ := NewEmoji(DefaultEmojiConfig()).Update(emojiHistoryMsg{"🍎", "not an emoji", "🎉"})

	tests := []struct {
		query        string
		wantGlyphs   []string
		wantSections []string
	}{
		{query: "", wantGlyphs: []string{"🍎", "🎉", "😀"}, wantSections: []string{"Recent", "Recent", "Smileys"}},
		{query: "   ", wantGlyphs: []string{"🍎", "🎉", "😀"}, wantSections: []string{"Recent", "Recent", "Smileys"}},
		{query: "red apple", wantGlyphs: []string{"🍎"}, wantSections: []string{"Food & drink"}},
	}

	for _, test := range tests {
		results := mode.SetQuery(test.query).(Emoji).results

		if len(results) < len(test.wantGlyphs) {
			t.Fatalf("query %q: %d results, want at least %d", test.query, len(results), len(test.wantGlyphs))
		}

		for i, entry := range results[:len(test.wantGlyphs)] {
			if entry.Glyph != test.wantGlyphs[i] || entry.Section != test.wantSections[i] {
				t.Errorf("query %q row %d = %q in %q, want %q in %q", test.query, i, entry.Glyph, entry.Section, test.wantGlyphs[i], test.wantSections[i])
			}
		}
	}
}

func TestEmojiActivateRecordsRecent(t *testing.T) {
	tests := []struct {
		name        string
		previous    []string
		query       string
		index       int
		wantCommand bool
		wantRecent  []string
	}{
		{name: "first pick", query: "eyes", index: 0, wantCommand: true, wantRecent: []string{"👀"}},
		{name: "repeat pick moves to front", previous: []string{"🍎", "👀"}, query: "eyes", index: 0, wantCommand: true, wantRecent: []string{"👀", "🍎"}},
		{name: "history capped at ten", previous: []string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}, query: "eyes", index: 0, wantCommand: true, wantRecent: []string{"👀", "0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣"}},
		{name: "index past the last row", query: "eyes", index: 1000, wantCommand: false},
		{name: "negative index", query: "eyes", index: -1, wantCommand: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("PATH", "")
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			t.Setenv("XDG_CACHE_HOME", t.TempDir())

			path, err := launtuiDataPath(emojiHistoryFile)

			if err != nil {
				t.Fatal(err)
			}

			err = saveJSON(path, test.previous)

			if err != nil {
				t.Fatal(err)
			}

			cmd := NewEmoji(DefaultEmojiConfig()).SetQuery(test.query).Activate(test.index)

			if (cmd != nil) != test.wantCommand {
				t.Fatalf("command returned = %v, want %v", cmd != nil, test.wantCommand)
			}

			if cmd == nil {
				return
			}

			if _, ok := cmd().(RequestQuitMsg); !ok {
				t.Fatal("activation should request quit")
			}

			recent, _ := loadJSON[[]string](path)

			if !slices.Equal(recent, test.wantRecent) {
				t.Fatalf("recent = %q, want %q", recent, test.wantRecent)
			}
		})
	}
}
