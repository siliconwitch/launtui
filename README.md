# launtui

A fast, keyboard-driven launcher for the terminal, with a clock and battery
readout at a glance. One search box, six modes:

- **Run** — fuzzy-search your desktop applications and launch them. Terminal
  apps (`Terminal=true` entries like btop) open in your terminal emulator.
- **Calc** — evaluate arithmetic (`(2+3)*4`, `2^10`), convert units
  (`5 miles to km`, `100 f to c`, `1 gib to mb`) and currencies
  (`10 gbp to usd`, live ECB rates cached for a day). Enter copies the result.
  Past calculations are kept; scroll down to one and press enter to copy its
  answer again.
- **Pass** — fuzzy-search your [pass](https://www.passwordstore.org) store.
  Enter prompts for your GPG passphrase in the terminal, copies the password
  to the clipboard, and saves the entry's second line (username/email) to the
  clipboard history. The password itself is never written to history.
- **Proj** — fuzzy-search your projects directory and open one in your editor.
  Git projects are fetched in the background and show their branch (green
  clean, red dirty) plus blue ↑/↓ arrows when there is anything to push or
  pull.
- **Clip** — clipboard history. Enter copies the selected entry back to the
  clipboard, ready to paste. Run `launtui -watch` in the background to record
  everything you copy.
- **Web** — anything that looks like a web address (`google.com`) offers to
  open in your browser, and any other query (`how do I update go`) falls back
  to a web search. Past visits and searches are kept; scroll down to one and
  press enter to open it again.

Built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss). Largely vibecoded —
written with AI assistance and reviewed by a human.

## Installation

### Arch Linux (AUR)

_Coming soon._

### Nix

_Coming soon._

### From source

1. Clone the repository:

    ```sh
    git clone https://github.com/siliconwitch/launtui.git
    cd launtui
    ```

1. Build the static binary:

    ```sh
    CGO_ENABLED=0 go build -o launtui .
    ```

1. Install it onto your `PATH`:

    ```sh
    sudo install -Dm755 launtui /usr/local/bin/launtui
    ```

## Dependencies

| Feature           | Requires                                                  |
| ----------------- | --------------------------------------------------------- |
| Build from source | [Go](https://go.dev) 1.26+                                |
| Battery icon      | A [Nerd Font](https://www.nerdfonts.com)                  |
| Clipboard         | wl-clipboard (Wayland) or xclip/xsel (X11)                |
| Passwords         | [pass](https://www.passwordstore.org) and gpg             |
| Projects          | git                                                       |
| Web               | xdg-utils (`xdg-open`)                                    |
| Currency rates    | Network access to frankfurter.dev (cached for 24 h)       |

## Usage

Run `launtui` in a terminal and start typing. It searches your apps by default
and switches mode automatically when your query fits another one better (e.g.
`4+5` jumps to the calculator, an unmatched question falls back to web search).
`Tab` and `Shift-Tab` step through the modes by hand. In the Calc, Clip and
Web modes, `Del` removes the selected history entry and `Ctrl-Del` clears that
mode's entire history. Press `Ctrl-h` for the full list of keybindings.

Start directly in a single mode (this turns off the automatic switching):

```sh
launtui -r   # Run
launtui -c   # Calculator
launtui -p   # Passwords
launtui -o   # Projects
launtui -v   # Clipboard
launtui -s   # Web search
```

### Clipboard watcher

The Clip mode records everything launtui itself copies. To also record copies
made anywhere else, keep the watcher running in the background — for example
in your compositor's autostart (sway/niri/hyprland):

```
exec launtui -watch
```

It uses `wl-paste --watch` on Wayland and falls back to polling on X11.
Passwords copied through the Pass mode are recognised and never recorded.

## Config

launtui reads `~/.config/launtui/config.toml` (honouring `$XDG_CONFIG_HOME`,
or an explicit `$LAUNTUI_CONFIG` path). Every key is optional and falls back
to the default below.

```toml
[run]
enabled  = true
exclude  = []                    # app names to hide, exactly as shown in the list
terminal = ""                    # terminal for Terminal=true apps ($TERMINAL or auto-detected)

[calculator]
enabled     = true
precision   = 6                  # max decimal places in the result
max_history = 50                 # calculations kept in history

[passwords]
enabled = true
store   = ""                     # password store path ($PASSWORD_STORE_DIR or ~/.password-store)

[projects]
enabled = true
dir     = "~/projects"           # directory containing your projects
editor  = ""                     # editor command ($VISUAL or $EDITOR when empty)

[clipboard]
enabled   = true
max_items = 100                  # clipboard entries kept in history

[web]
enabled     = true
search_url  = "https://duckduckgo.com/?q=%s"   # %s is the escaped query
max_history = 50                 # visits and searches kept in history

[clock]
enabled = true
format  = "Mon 2 Jan - 15:04"   # Go reference-time layout

[battery]
enabled = true
device  = "BAT0"                 # name under /sys/class/power_supply

[help]
enabled = true
```

## Contributing

Contributions are welcome — open an issue or a pull request. launtui is largely
vibecoded, and you're welcome to contribute with AI assistance too. Just read
and test your code before submitting.

### Porting (macOS and others)

launtui currently targets Linux, but it already cross-compiles for macOS and
the OS-specific behaviour is confined to a handful of functions. A port needs:

- `internal/widgets/system.go` — add `pbcopy`/`pbpaste` to the clipboard tool
  lists.
- `internal/widgets/web.go` — launch URLs with `open` instead of `xdg-open`.
- `internal/widgets/run.go` — `scanDesktopApps` and `launchArgv` are XDG
  desktop-entry based; macOS needs an `.app` bundle scanner and `open -a`.
- `internal/widgets/battery.go` — reads `/sys/class/power_supply`; on other
  platforms the widget silently hides itself, so this is optional (`pmset` on
  macOS).

Everything else (passwords, projects, calculator, clipboard history) is
portable as is.

## License

[MIT](LICENSE) © Raj Nakarja
