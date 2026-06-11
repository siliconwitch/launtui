package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/siliconwitch/launtui/internal/tui"
	"github.com/siliconwitch/launtui/internal/widgets"
)

func main() {
	run := flag.Bool("r", false, "start in Run mode")
	calculator := flag.Bool("c", false, "start in Calculator mode")
	passwords := flag.Bool("p", false, "start in Passwords mode")
	onepassword := flag.Bool("1", false, "start in 1Password mode")
	projects := flag.Bool("o", false, "start in Projects mode")
	clipboard := flag.Bool("v", false, "start in Clipboard mode")
	web := flag.Bool("s", false, "start in Web search mode")
	watch := flag.Bool("watch", false, "watch the clipboard and record history")
	record := flag.Bool("record", false, "record stdin into clipboard history")

	flag.Parse()

	if *watch || *record {
		runClipboardTool(*watch)

		return
	}

	startLetter := ""

	switch {
	case *run:
		startLetter = "r"
	case *calculator:
		startLetter = "c"
	case *passwords:
		startLetter = "p"
	case *onepassword:
		startLetter = "1"
	case *projects:
		startLetter = "o"
	case *clipboard:
		startLetter = "v"
	case *web:
		startLetter = "s"
	}

	startHotkey := ""

	if startLetter != "" {
		startHotkey = "ctrl+" + startLetter
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

func runClipboardTool(watch bool) {
	cfg := widgets.DefaultClipboardConfig()

	err := tui.Load(&cfg)

	if err != nil {
		fmt.Fprintln(os.Stderr, "launtui: config:", err)
	}

	if watch {
		err = widgets.WatchClipboard(cfg)
	} else {
		err = widgets.RecordClipboardStdin(cfg)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "launtui:", err)
		os.Exit(1)
	}
}
