package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/siliconwitch/launtui/internal/tui"
)

func main() {
	app, err := tui.New()

	if err != nil {
		fmt.Fprintln(os.Stderr, "launtui: config:", err)
	}

	_, err = tea.NewProgram(app, tea.WithAltScreen()).Run()

	if err != nil {
		fmt.Fprintln(os.Stderr, "launtui:", err)
		os.Exit(1)
	}
}
