//go:build !darwin

package widgets

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) {
	tools := [][]string{
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
	}

	for _, tool := range tools {
		path, err := exec.LookPath(tool[0])

		if err != nil {
			continue
		}

		cmd := exec.Command(path, tool[1:]...)
		cmd.Stdin = strings.NewReader(text)

		if cmd.Run() == nil {
			return
		}
	}
}

func readClipboard() string {
	tools := [][]string{
		{"wl-paste", "--no-newline", "--type", "text"},
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
	}

	for _, tool := range tools {
		path, err := exec.LookPath(tool[0])

		if err != nil {
			continue
		}

		output, err := exec.Command(path, tool[1:]...).Output()

		if err != nil {
			continue
		}

		return string(output)
	}

	return ""
}

func openURL(address string) {
	spawnDetached("xdg-open", address)
}
