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
	// -sequence is internal: the passwords widget re-invokes launtui as
	// `launtui -sequence` with "username\npassword" piped to its stdin, and
	// this process serves the staged clipboard hand-off (see RunPasteSequence).
	// It is never typed by a user, so it is matched here directly and left
	// unregistered as a flag: that keeps it out of -help, which also means flag
	// parsing must be skipped for it.
	sequence := len(os.Args) == 2 && os.Args[1] == "-sequence"

	modeFlags := []struct {
		letter string
		name   string
	}{
		{"r", "Run"},
		{"c", "Calculator"},
		{"p", "Passwords"},
		{"o", "Projects"},
		{"v", "Clipboard"},
		{"e", "Emoji"},
		{"s", "Web search"},
	}

	selected := make([]*bool, len(modeFlags))

	for i, mode := range modeFlags {
		selected[i] = flag.Bool(mode.letter, false, "start in "+mode.name+" mode")
	}

	watch := flag.Bool("watch", false, "watch the clipboard and record history")

	if !sequence {
		flag.Parse()
	}

	if sequence {
		if err := widgets.RunPasteSequence(); err != nil {
			fmt.Fprintln(os.Stderr, "launtui:", err)
			os.Exit(1)
		}

		return
	}

	if *watch {
		clipboardConfig := widgets.DefaultClipboardConfig()

		if err := tui.LoadConfig(&clipboardConfig); err != nil {
			fmt.Fprintln(os.Stderr, "launtui: config:", err)
		}

		if err := widgets.WatchClipboard(clipboardConfig); err != nil {
			fmt.Fprintln(os.Stderr, "launtui:", err)
			os.Exit(1)
		}

		return
	}

	startHotkey := ""

	for i, mode := range modeFlags {
		if *selected[i] {
			startHotkey = "ctrl+" + mode.letter

			break
		}
	}

	app, _ := tui.New(startHotkey)

	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "launtui:", err)
		os.Exit(1)
	}
}
