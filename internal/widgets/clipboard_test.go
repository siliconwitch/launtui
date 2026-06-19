package widgets

import (
	"strings"
	"testing"
	"unicode"
)

func TestClipboardPreview(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{name: "plain text", text: "hello world", want: "hello world"},
		{name: "first non-empty line", text: "\n\n  first\nsecond", want: "first"},
		{name: "tabs become spaces", text: "Datasheet\tF.Fab\t0.2 mm", want: "Datasheet F.Fab 0.2 mm"},
		{name: "carriage return in middle", text: "before\rafter", want: "before after"},
		{name: "escape sequence is neutralized", text: "red\x1b[31mtext", want: "red [31mtext"},
		{name: "trailing control trimmed", text: "value\t\r", want: "value"},
	}

	for _, test := range cases {
		got := clipboardPreview(test.text)

		if got != test.want {
			t.Errorf("%s: clipboardPreview(%q) = %q, want %q", test.name, test.text, got, test.want)
		}

		if strings.IndexFunc(got, unicode.IsControl) != -1 {
			t.Errorf("%s: preview still contains a control character: %q", test.name, got)
		}
	}
}
