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
	// -record is internal: the clipboard watcher re-invokes launtui as
	// `launtui -record` once per clipboard change and pipes the new contents to
	// its stdin (see WatchClipboard). It is never typed by a user, so it is
	// matched here directly and left unregistered as a flag — keeping it out of
	// -help, which also means flag parsing must be skipped for it.
	record := len(os.Args) == 2 && os.Args[1] == "-record"

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

	if !record {
		flag.Parse()
	}

	if record || *watch {
		clipboardConfig := widgets.DefaultClipboardConfig()

		if err := tui.LoadConfig(&clipboardConfig); err != nil {
			fmt.Fprintln(os.Stderr, "launtui: config:", err)
		}

		var err error

		if record {
			err = widgets.RecordClipboardStdin(clipboardConfig)
		} else {
			err = widgets.WatchClipboard(clipboardConfig)
		}

		if err != nil {
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
