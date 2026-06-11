# launtui

A fast, keyboard-driven launcher for the terminal, with a clock and battery
readout at a glance. Runs on Linux and macOS. One search box, eight modes
(the platform-specific ones appear only where they apply):

- **Run** — fuzzy-search your applications and launch them. On Linux this reads
  your `.desktop` files (terminal apps like btop open in your terminal
  emulator); on macOS it scans `/Applications`, `/System/Applications` and
  `~/Applications` and opens the bundle.
- **Calc** — evaluate arithmetic (`(2+3)*4`, `2^10`), convert units
  (`5 miles to km`, `100 f to c`, `1 gib to mb`) and currencies
  (`10 gbp to usd`, live ECB rates cached for a day). Enter copies the result.
  Past calculations are kept; scroll down to one and press enter to copy its
  answer again.
- **Pass** — fuzzy-search your [pass](https://www.passwordstore.org) store.
  Enter prompts for your GPG passphrase in the terminal, copies the password
  to the clipboard, and saves the entry's second line (username/email) to the
  clipboard history. The password itself is never written to history.
- **1Pass** — fuzzy-search your [1Password](https://1password.com) logins via
  the [`op`](https://developer.1password.com/docs/cli/) CLI (macOS or Linux).
  Searches every signed-in account by default; Enter copies the password to the
  clipboard and saves the username to clipboard history, mirroring Pass. This
  mode is hidden unless `op` is installed.
- **Proj** — fuzzy-search your projects directory and open one in your editor.
  Git projects are fetched in the background and show their branch (green
  clean, yellow dirty) plus pending pushes/pulls as ↑/↓ counts.
- **Clip** — clipboard history. Enter copies the selected entry back to the
  clipboard, ready to paste. Run `launtui -watch` in the background to record
  everything you copy.
- **Web** — anything that looks like a web address (`google.com`) offers to
  open in your browser, and any other query (`how do I update go`) falls back
  to a web search.
- **Safari** — (macOS only) fuzzy-search your open Safari tabs, bookmarks and
  recent history in one list. Enter switches to a matching tab, or opens a
  bookmark/history page. Needs Full Disk Access (bookmarks, history) and
  Automation (tabs) granted to your terminal; hidden on non-macOS systems.

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
    sudo install -Dm755 launtui /usr/local/bin/launtui   # Linux
    ```

    On macOS:

    ```sh
    install -m755 launtui /usr/local/bin/launtui
    ```

    The Safari mode reads bookmarks and history from your Safari data and lists
    open tabs, so the first time you use it macOS will ask you to grant your
    terminal Full Disk Access (bookmarks, history) and Automation (tabs) under
    System Settings > Privacy & Security.

## Dependencies

| Feature           | Requires                                                       |
| ----------------- | -------------------------------------------------------------- |
| Build from source | [Go](https://go.dev) 1.26+                                     |
| Battery icon      | A [Nerd Font](https://www.nerdfonts.com)                       |
| Battery readout   | Linux: `/sys` power supply. macOS: `pmset` (built in)          |
| Clipboard         | Linux: wl-clipboard or xclip/xsel. macOS: `pbcopy`/`pbpaste`   |
| Passwords         | [pass](https://www.passwordstore.org) and gpg                  |
| 1Password         | [`op`](https://developer.1password.com/docs/cli/) CLI          |
| Projects          | git                                                            |
| Web               | Linux: xdg-utils (`xdg-open`). macOS: `open` (built in)        |
| Safari            | macOS, plus Full Disk Access and Automation grants             |
| Currency rates    | Network access to frankfurter.dev (cached for 24 h)            |

## Usage

Run `launtui` in a terminal and start typing. It searches your apps by default
and switches mode automatically when your query fits another one better (e.g.
`4+5` jumps to the calculator, an unmatched question falls back to web search).
Press `Ctrl-h` for the full list of keybindings.

Start directly in a single mode (this turns off the automatic switching):

```sh
launtui -r   # Run
launtui -c   # Calculator
launtui -p   # Passwords
launtui -1   # 1Password
launtui -o   # Projects
launtui -v   # Clipboard
launtui -s   # Web search
launtui -b   # Safari
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
or an explicit `$LAUNTUI_CONFIG` path) on both Linux and macOS. Every key is
optional and falls back to the default below.

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

[onepassword]
enabled = true                   # auto-hidden unless the `op` CLI is installed
account = ""                     # empty = all signed-in accounts; a UUID/email narrows it

[projects]
enabled = true
dir     = "~/projects"           # directory containing your projects
editor  = ""                     # editor command ($VISUAL or $EDITOR when empty)

[clipboard]
enabled   = true
max_items = 100                  # clipboard entries kept in history

[web]
enabled    = true
search_url = "https://duckduckgo.com/?q=%s"   # %s is the escaped query

[safari]
enabled       = true             # macOS only; tabs, bookmarks and history
history_limit = 2000             # most recent history entries to search

[clock]
enabled = true
format  = "Mon 2 Jan - 15:04"   # Go reference-time layout

[battery]
enabled = true
device  = "BAT0"                 # name under /sys/class/power_supply (Linux only; ignored on macOS)

[help]
enabled = true
```

## Contributing

Contributions are welcome — open an issue or a pull request. launtui is largely
vibecoded, and you're welcome to contribute with AI assistance too. Just read
and test your code before submitting.

## License

[MIT](LICENSE) © Raj Nakarja
