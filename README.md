# launtui

A fast, keyboard-driven application launcher for the terminal, with a clock and
battery readout at a glance.

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

| Feature           | Requires                                       |
| ----------------- | ---------------------------------------------- |
| Build from source | [Go](https://go.dev) 1.26+                     |
| Battery icon      | A [Nerd Font](https://www.nerdfonts.com)       |

## Usage

Run `launtui` in a terminal and start typing. It searches your apps by default
and switches mode automatically when your query fits another one better (e.g.
`4+5` jumps to the calculator). Press `Ctrl-h` for the full list of keybindings.

Start directly in a single mode (this turns off the automatic switching):

```sh
launtui -r   # Run
launtui -c   # Calculator
launtui -v   # Clipboard
launtui -p   # Passwords
```

## Config

launtui reads `~/.config/launtui/config.toml` (honouring `$XDG_CONFIG_HOME`).
Every key is optional and falls back to the default below.

```toml
[run]
enabled = true

[calculator]
enabled   = true
precision = 6                    # max decimal places in the result

[clipboard]
enabled = true

[passwords]
enabled = true

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

## License

[MIT](LICENSE) © Raj Nakarja
