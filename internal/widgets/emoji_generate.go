//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type annotation struct {
	Default []string `json:"default"`
	Speech  []string `json:"tts"`
}

type emoji struct {
	Glyph      string   `json:"glyph"`
	Name       string   `json:"name"`
	Section    string   `json:"section"`
	NameWords  []string `json:"nameWords"`
	GroupWords []string `json:"groupWords"`
	Keywords   []string `json:"keywords"`
}

func download(url string) []byte {
	client := http.Client{Timeout: time.Minute}

	response, err := client.Get(url)

	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatalf("%s: %s", url, response.Status)
	}

	body, err := io.ReadAll(response.Body)

	if err != nil {
		log.Fatal(err)
	}

	return body
}

func words(texts ...string) []string {
	var found []string

	for _, text := range texts {
		folded, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn))), strings.ToLower(text))

		if err != nil {
			log.Fatal(err)
		}

		found = append(found, strings.FieldsFunc(folded, func(character rune) bool {
			return !unicode.IsLetter(character) && !unicode.IsNumber(character)
		})...)
	}

	slices.Sort(found)

	return slices.Compact(found)
}

func main() {
	license := download("https://raw.githubusercontent.com/unicode-org/cldr-json/46.0.0/LICENSE")
	emojiTest := download("https://unicode.org/Public/emoji/16.0/emoji-test.txt")

	var annotations struct {
		Annotations struct {
			Annotations map[string]annotation `json:"annotations"`
		} `json:"annotations"`
	}

	err := json.Unmarshal(download("https://raw.githubusercontent.com/unicode-org/cldr-json/46.0.0/cldr-json/cldr-annotations-full/annotations/en/annotations.json"), &annotations)

	if err != nil {
		log.Fatal(err)
	}

	var derived struct {
		AnnotationsDerived struct {
			Annotations map[string]annotation `json:"annotations"`
		} `json:"annotationsDerived"`
	}

	err = json.Unmarshal(download("https://raw.githubusercontent.com/unicode-org/cldr-json/46.0.0/cldr-json/cldr-annotations-derived-full/annotationsDerived/en/annotations.json"), &derived)

	if err != nil {
		log.Fatal(err)
	}

	sectionOrder := []string{
		"Smileys", "People", "Relationships", "Clothes", "Animals", "Plants", "Sky & weather",
		"Food & drink", "Travel & places", "Activities", "Objects", "Symbols", "Flags",
	}

	var group, subgroup string
	var catalogue []emoji

	for _, line := range strings.Split(string(emojiTest), "\n") {
		if heading, ok := strings.CutPrefix(line, "# group: "); ok {
			group = heading

			continue
		}

		if heading, ok := strings.CutPrefix(line, "# subgroup: "); ok {
			subgroup = heading

			continue
		}

		codepoints, rest, ok := strings.Cut(line, ";")

		if !ok || strings.HasPrefix(line, "#") {
			continue
		}

		status, description, ok := strings.Cut(rest, "#")

		if !ok || strings.TrimSpace(status) != "fully-qualified" || group == "Component" {
			continue
		}

		var glyph strings.Builder
		skinToned := false

		for _, field := range strings.Fields(codepoints) {
			codepoint, err := strconv.ParseUint(field, 16, 32)

			if err != nil {
				log.Fatal(err)
			}

			skinToned = skinToned || (codepoint >= 0x1F3FB && codepoint <= 0x1F3FF)
			glyph.WriteRune(rune(codepoint))
		}

		if skinToned {
			continue
		}

		descriptionFields := strings.Fields(description)

		if len(descriptionFields) < 3 {
			log.Fatalf("malformed line: %q", line)
		}

		section := group

		switch {
		case group == "Smileys & Emotion":
			section = "Smileys"
		case subgroup == "family":
			section = "Relationships"
		case group == "People & Body":
			section = "People"
		case subgroup == "clothing":
			section = "Clothes"
		case strings.HasPrefix(subgroup, "animal-"):
			section = "Animals"
		case strings.HasPrefix(subgroup, "plant-"):
			section = "Plants"
		case subgroup == "sky & weather":
			section = "Sky & weather"
		case group == "Food & Drink":
			section = "Food & drink"
		case group == "Travel & Places":
			section = "Travel & places"
		}

		if !slices.Contains(sectionOrder, section) {
			log.Fatalf("no section order for %q", section)
		}

		name := strings.Join(descriptionFields[2:], " ")
		annotationKey := strings.ReplaceAll(glyph.String(), "️", "")

		cldr := annotations.Annotations.Annotations[annotationKey]
		cldrDerived := derived.AnnotationsDerived.Annotations[annotationKey]

		keywordSources := slices.Concat([]string{name, subgroup}, cldr.Default, cldr.Speech, cldrDerived.Default, cldrDerived.Speech)

		catalogue = append(catalogue, emoji{
			Glyph:      glyph.String(),
			Name:       name,
			Section:    section,
			NameWords:  words(name),
			GroupWords: words(section, group),
			Keywords:   words(keywordSources...),
		})
	}

	slices.SortStableFunc(catalogue, func(first, second emoji) int {
		return slices.Index(sectionOrder, first.Section) - slices.Index(sectionOrder, second.Section)
	})

	var line bytes.Buffer
	encoder := json.NewEncoder(&line)
	encoder.SetEscapeHTML(false)

	encode := func(value any) string {
		line.Reset()

		err := encoder.Encode(value)

		if err != nil {
			log.Fatal(err)
		}

		return strings.TrimSuffix(line.String(), "\n")
	}

	entries := make([]string, len(catalogue))

	for index, entry := range catalogue {
		entries[index] = encode(entry)
	}

	output := "{\"license\":" + encode(string(license)) + ",\"emoji\":[\n" + strings.Join(entries, ",\n") + "\n]}\n"

	err = os.WriteFile("emoji.json", []byte(output), 0o644)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("wrote %d emoji\n", len(catalogue))
}
