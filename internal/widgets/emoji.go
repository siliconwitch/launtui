package widgets

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type EmojiConfig struct {
	Enabled bool `toml:"enabled"`
}

func (EmojiConfig) SectionName() string { return "emoji" }

func DefaultEmojiConfig() EmojiConfig {
	return EmojiConfig{Enabled: true}
}

//go:generate go run emoji_generate.go

//go:embed emoji.json
var emojiCatalogue []byte

const (
	emojiHistoryFile   = "emoji-history.json"
	emojiRecentSection = "Recent"
)

var emojiAccent = lipgloss.Color("3")

type emojiEntry struct {
	Glyph      string   `json:"glyph"`
	Name       string   `json:"name"`
	Section    string   `json:"section"`
	NameWords  []string `json:"nameWords"`
	GroupWords []string `json:"groupWords"`
	Keywords   []string `json:"keywords"`
}

const (
	emojiNameMatch = iota
	emojiSectionMatch
	emojiNameWordMatch
	emojiKeywordMatch
	emojiSectionPrefix
	emojiNameWordPrefix
	emojiKeywordPrefix
	emojiUnmatched
)

type emojiHistoryMsg []string

type Emoji struct {
	config        EmojiConfig
	entries       []emojiEntry
	sectionLabels map[string]string
	recent        []string
	query         string
	results       []emojiEntry
}

func NewEmoji(config EmojiConfig) Emoji {
	if !config.Enabled {
		return Emoji{config: config}
	}

	var catalogue struct {
		Emoji []emojiEntry `json:"emoji"`
	}

	_ = json.Unmarshal(emojiCatalogue, &catalogue)

	sectionLabels := map[string]string{emojiRecentSection: subtleStyle.Render(emojiRecentSection)}

	for _, entry := range catalogue.Emoji {
		if _, ok := sectionLabels[entry.Section]; !ok {
			sectionLabels[entry.Section] = subtleStyle.Render(entry.Section)
		}
	}

	return Emoji{config: config, entries: catalogue.Emoji, sectionLabels: sectionLabels, results: catalogue.Emoji}
}

func (Emoji) Name() string    { return "Emoji" }
func (Emoji) Hotkey() string  { return "ctrl+e" }
func (e Emoji) Enabled() bool { return e.config.Enabled }

func (e Emoji) Init() tea.Cmd {
	if !e.config.Enabled {
		return nil
	}

	return func() tea.Msg {
		path, err := launtuiDataPath(emojiHistoryFile)

		if err != nil {
			return emojiHistoryMsg(nil)
		}

		history, _ := loadJSON[[]string](path)

		return emojiHistoryMsg(history)
	}
}

func (e Emoji) Update(msg tea.Msg) (Mode, tea.Cmd) {
	history, ok := msg.(emojiHistoryMsg)

	if !ok {
		return e, nil
	}

	e.recent = history
	e.results = emojiResults(e.entries, e.recent, e.query)

	return e, nil
}

func (e Emoji) SetQuery(query string) Mode {
	e.query = query
	e.results = emojiResults(e.entries, e.recent, e.query)

	return e
}

func emojiResults(entries []emojiEntry, recent []string, query string) []emojiEntry {
	trimmedQuery := strings.TrimSpace(query)

	if trimmedQuery == "" {
		results := make([]emojiEntry, 0, len(recent)+len(entries))

		for _, glyph := range recent {
			index := slices.IndexFunc(entries, func(entry emojiEntry) bool { return entry.Glyph == glyph })

			if index < 0 {
				continue
			}

			entry := entries[index]
			entry.Section = emojiRecentSection
			results = append(results, entry)
		}

		return append(results, entries...)
	}

	foldedQuery, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn))), strings.ToLower(trimmedQuery))

	if err != nil {
		foldedQuery = strings.ToLower(trimmedQuery)
	}

	queryWords := strings.FieldsFunc(foldedQuery, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})

	queryName := strings.Join(slices.Compact(slices.Sorted(slices.Values(queryWords))), " ")

	ranked := make([][]emojiEntry, emojiUnmatched)
	var unmatched []emojiEntry

	for _, entry := range entries {
		if strings.Join(entry.NameWords, " ") == queryName {
			ranked[emojiNameMatch] = append(ranked[emojiNameMatch], entry)

			continue
		}

		rank := emojiUnmatched

		if len(queryWords) > 0 {
			rank = emojiSectionMatch
		}

		for _, word := range queryWords {
			startsWithWord := func(candidate string) bool { return strings.HasPrefix(candidate, word) }
			wordRank := emojiUnmatched

			switch {
			case slices.Contains(entry.GroupWords, word):
				wordRank = emojiSectionMatch
			case slices.Contains(entry.NameWords, word):
				wordRank = emojiNameWordMatch
			case slices.Contains(entry.Keywords, word):
				wordRank = emojiKeywordMatch
			case slices.ContainsFunc(entry.GroupWords, startsWithWord):
				wordRank = emojiSectionPrefix
			case slices.ContainsFunc(entry.NameWords, startsWithWord):
				wordRank = emojiNameWordPrefix
			case slices.ContainsFunc(entry.Keywords, startsWithWord):
				wordRank = emojiKeywordPrefix
			}

			rank = max(rank, wordRank)
		}

		if rank < emojiUnmatched {
			ranked[rank] = append(ranked[rank], entry)

			continue
		}

		unmatched = append(unmatched, entry)
	}

	for rank, rankedEntries := range ranked {
		if rank == emojiSectionMatch || rank == emojiSectionPrefix {
			continue
		}

		slices.SortStableFunc(rankedEntries, func(first, second emojiEntry) int {
			return len(first.NameWords) - len(second.NameWords)
		})
	}

	names := make([]string, len(unmatched))

	for i, entry := range unmatched {
		names[i] = entry.Name
	}

	var fuzzyMatches []emojiEntry

	for _, match := range fuzzy.Find(trimmedQuery, names) {
		fuzzyMatches = append(fuzzyMatches, unmatched[match.Index])
	}

	return slices.Concat(append(ranked, fuzzyMatches)...)
}

func (Emoji) Accent() lipgloss.Color { return emojiAccent }

func (e Emoji) Status() string {
	if len(e.results) == 0 {
		return subtleStyle.Render("no matching emoji")
	}

	return ""
}

func (e Emoji) Rows() []Row {
	rows := make([]Row, len(e.results))

	for i, entry := range e.results {
		rows[i] = Row{left: entry.Glyph + "  " + entry.Name, right: e.sectionLabels[entry.Section]}
	}

	return rows
}

func (e Emoji) Activate(index int) tea.Cmd {
	if index < 0 || index >= len(e.results) {
		return nil
	}

	glyph := e.results[index].Glyph

	return func() tea.Msg {
		copyToClipboard(glyph)
		recordClipboardText(glyph, 0)

		path, err := launtuiDataPath(emojiHistoryFile)

		if err == nil {
			previous, _ := loadJSON[[]string](path)

			recent := prependCapped(previous, glyph, 10, func(existing string) bool {
				return existing == glyph
			})

			_ = saveJSON(path, recent)
		}

		return RequestQuitMsg{}
	}
}
