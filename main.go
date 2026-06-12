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
	modeFlags := []struct {
		letter string
		name   string
	}{
		{"r", "Run"},
		{"c", "Calculator"},
		{"p", "Passwords"},
		{"o", "Projects"},
		{"v", "Clipboard"},
		{"s", "Web search"},
	}

	selected := make([]*bool, len(modeFlags))

	for i, mode := range modeFlags {
		selected[i] = flag.Bool(mode.letter, false, "start in "+mode.name+" mode")
	}

	watch := flag.Bool("watch", false, "watch the clipboard and record history")
	record := flag.Bool("record", false, "record stdin into clipboard history")

	flag.Parse()

	if *watch || *record {
		runClipboardTool(*watch)

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
