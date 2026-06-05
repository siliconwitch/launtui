package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/siliconwitch/launtui/internal/tui"
)

func main() {
	run := flag.Bool("r", false, "start in Run mode")
	calculator := flag.Bool("c", false, "start in Calculator mode")
	clipboard := flag.Bool("v", false, "start in Clipboard mode")
	passwords := flag.Bool("p", false, "start in Passwords mode")

	flag.Parse()

	startHotkey := ""

	switch {
	case *run:
		startHotkey = "ctrl+r"
	case *calculator:
		startHotkey = "ctrl+c"
	case *clipboard:
		startHotkey = "ctrl+v"
	case *passwords:
		startHotkey = "ctrl+p"
	}

	app, err := tui.New(startHotkey)

	if err != nil {
		fmt.Fprintln(os.Stderr, "launtui: config:", err)
	}

	_, err = tea.NewProgram(app, tea.WithAltScreen()).Run()

	if err != nil {
		fmt.Fprintln(os.Stderr, "launtui:", err)
		os.Exit(1)
	}
}
