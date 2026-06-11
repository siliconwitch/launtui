//go:build darwin

package widgets

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)

	_ = cmd.Run()
}

func readClipboard() string {
	output, err := exec.Command("pbpaste").Output()

	if err != nil {
		return ""
	}

	return string(output)
}

func openURL(address string) {
	spawnDetached("open", address)
}
