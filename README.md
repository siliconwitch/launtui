# launtui

A fast TUI run launcher, calculator, file finder and more — one keypress away.

Built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea) +
[Lip Gloss](https://github.com/charmbracelet/lipgloss), styled after
[bluetui](https://github.com/pythops/bluetui) and
[impala](https://github.com/pythops/impala).

## Modes

Switch sections with `Tab` / `Shift+Tab`:

- **Run** — fuzzy-find an executable on `$PATH` and launch it detached.
- **Calc** — type an arithmetic expression and see the result live
  (`+ - * / % ^`, parentheses, unary minus). `Enter` prints the result.
- **Files** — browse your home directory and open an entry with the system
  handler (`xdg-open`).

## Keys

| Key             | Action               |
| --------------- | -------------------- |
| type            | filter               |
| `↑/↓`           | move selection       |
| `Enter`         | activate             |
| `Tab` / `S-Tab` | next / previous mode |
| `Esc` / `C-c`   | quit                 |

## Build & run

```sh
CGO_ENABLED=0 go build -o launtui .
./launtui
```

`CGO_ENABLED=0` produces a fully static binary with no system dependencies.

## Config

launtui reads `~/.config/launtui/config.toml` (honouring `$XDG_CONFIG_HOME`),
falling back to the built-in defaults when the file — or any individual key — is
absent. Every setting, with its description, options and default:

```toml
[launcher]
placeholder = "Search…"    # text shown in the empty search box

[calculator]
precision = 2              # digits after the decimal point — integer, 0 or more
angle     = "rad"          # unit for trig functions — "rad" or "deg"

[clock]
format = "15:04:05"        # Go reference-time layout, e.g. "15:04" or "Mon 2 Jan 15:04"

[clipboard]
max_entries = 50           # clipboard history entries to keep — integer, 1 or more
```

## Bind to a hotkey in niri

launtui is a terminal app, so spawn it in a small floating terminal from your
`config.kdl`. Example with `foot`:

```kdl
binds {
    Mod+Space { spawn "foot" "-a" "launtui" "launtui"; }
}

window-rule {
    match app-id="launtui"
    open-floating true
    default-column-width { fixed 720; }
    default-window-height { fixed 480; }
}
```

(Adjust the terminal command and window-rule to taste — `-a launtui` sets the
app-id the rule matches on.)
