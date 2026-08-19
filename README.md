# launtui

![launtui demo](docs/demo.gif)

A fast, keyboard-driven launcher for the terminal - one search box, with a clock
and battery at a glance. Start typing and it switches to whichever mode fits.

- **Run** - fuzzy-launch your desktop applications
- **Calc** - math (`sqrt`, `pi`, `2^10`), bitwise (`&`, `|`, `<<`, `xor`),
  number-base (`255 to hex`), unit, and currency conversion
- **Pass** - search your [pass](https://www.passwordstore.org) store, copy via
  GPG - the first paste gives the username, the second the password
- **Proj** - open a project in your editor, with live git status
- **Clip** - clipboard history
- **Emoji** - emoji picker
- **Web** - open a URL or search the web

Built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss). A single static binary
that shells out to the tools you already have.

## Install

### Arch

From the [AUR](https://aur.archlinux.org/packages/launtui-bin), with your
favourite helper:

```sh
paru -S launtui-bin
```

### Nix

Try it without installing:

```sh
nix run github:siliconwitch/launtui
```

Or add the flake as an input and take `packages.default`. On NixOS, the flake
also ships a module that installs launtui and runs the clipboard watcher as a
systemd user service:

```nix
{
  inputs.launtui.url = "github:siliconwitch/launtui";

  # In your system configuration:
  imports = [ launtui.nixosModules.default ];
  services.launtui.enable = true;
}
```

### Prebuilt binary

Every release ships static `amd64` and `arm64` tarballs. Grab one from the
[releases page](https://github.com/siliconwitch/launtui/releases):

```sh
tar xzf launtui_*_linux_amd64.tar.gz
sudo install -Dm755 launtui /usr/local/bin/launtui
```

### From source

Requires [Go](https://go.dev) 1.25+.

```sh
git clone https://github.com/siliconwitch/launtui.git
cd launtui
CGO_ENABLED=0 go build -o launtui .
sudo install -Dm755 launtui /usr/local/bin/launtui
```

The AUR and Nix packages are maintained upstream; if you use another
distribution and are willing to package and maintain launtui for it, you are
very welcome to. Open an issue and the package will be linked here.

Both manual installs can also add launtui to desktop menus and launchers; the
release tarball and the repository carry the same desktop entry:

```sh
sudo install -Dm644 contrib/desktop/launtui.desktop /usr/share/applications/launtui.desktop
```

## Dependencies

The binary is static and needs nothing at runtime. Each mode shells out to the
tool it fronts, and a mode whose tool is missing simply finds nothing.

| Feature           | Requires                                                  |
| ----------------- | --------------------------------------------------------- |
| Build from source | [Go](https://go.dev) 1.25+                                |
| Battery icon      | A [Nerd Font](https://www.nerdfonts.com)                  |
| Emoji glyphs      | A colour emoji font (e.g. Noto Color Emoji)               |
| Clipboard         | wl-clipboard (Wayland) or xclip/xsel (X11)                |
| Passwords         | [pass](https://www.passwordstore.org) and gpg             |
| Projects          | git                                                       |
| Web               | xdg-utils (`xdg-open`)                                    |
| Currency rates    | Network access to api.frankfurter.dev (cached for 24 h)   |

## Usage

Run `launtui` in a terminal and start typing. It searches your apps by default
and switches mode automatically when your query fits another one better (e.g.
`4+5` jumps to the calculator, an unmatched question falls back to web search).
`Tab` and `Shift-Tab` step through the modes by hand. In the Calc, Clip and
Web modes, `Del` removes the selected history entry and `Alt-Del` clears that
mode's entire history. Press `Ctrl-h` for the full list of keybindings.

Start directly in a single mode (this turns off the automatic switching):

```sh
launtui -r   # Run
launtui -c   # Calculator
launtui -p   # Passwords
launtui -o   # Projects
launtui -v   # Clipboard
launtui -e   # Emoji
launtui -s   # Web search
```

`launtui -version` prints the version, and `launtui -help` lists every flag.

launtui is meant to be opened by a hotkey. Bind one in your compositor to run
it in a small floating terminal, on sway for example:

```
bindsym $mod+space exec foot --app-id=launtui launtui
```

### Passwords

Activating a pass entry hands the credentials over in stages: the first paste
inserts the entry's username (its second line, when present) and every paste
after that inserts the password, until the clipboard is cleared 45 seconds
after the password is armed. A stage left unpasted for 45 seconds ends the
sequence and clears the clipboard, and copying anything else cancels the
remaining stages. Staged passwords never reach the clipboard history.
Staging needs a Wayland session and wl-clipboard 2.3+; elsewhere the password
is copied plainly and its recording is suppressed for the next 5 minutes.

### Clipboard watcher

The Clip mode records everything launtui itself copies. To also record copies
made anywhere else, keep `launtui -watch` running in the background. It polls
once a second. On Wayland it inspects the offered clipboard types before
reading anything, so sensitive copies - the Pass mode's staged pastes, or
anything else marked with `x-kde-passwordManagerHint` - are passed over without
being read or recorded.

Start it from your compositor's autostart (sway/niri/hyprland):

```
exec launtui -watch
```

Or run it as a systemd user service. The Arch package and the NixOS module
already install the unit, so enabling it is all that is left:

```sh
systemctl --user enable --now launtui-watch.service
```

Installing by hand instead? The release tarball and the repository both carry
[`contrib/systemd/launtui-watch.service`](contrib/systemd/launtui-watch.service),
which expects the binary at `/usr/local/bin/launtui`:

```sh
install -Dm644 contrib/systemd/launtui-watch.service \
    ~/.config/systemd/user/launtui-watch.service
systemctl --user daemon-reload
systemctl --user enable --now launtui-watch.service
```

The unit starts with `graphical-session.target`, so your compositor has to
reach that target and export `WAYLAND_DISPLAY` into the systemd user
environment. The watcher decides once at startup whether it can see a Wayland
clipboard, and one that starts without `WAYLAND_DISPLAY` cannot skip sensitive
copies. Restart the watcher after upgrading launtui.

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
enabled  = true
dirs     = ["~/Documents"]       # directories scanned for projects
projects = []                    # extra project paths to include explicitly
editor   = ""                    # editor command ($VISUAL or $EDITOR when empty)
terminal = ""                    # terminal for terminal-based editors ($TERMINAL or auto-detected)
# editor_terminal = true         # force terminal vs GUI launch (auto-detected when unset)

[clipboard]
enabled     = true
max_history = 50                 # entries kept in history

[web]
enabled     = true
search_url  = "https://duckduckgo.com/?q=%s"   # %s is the escaped query
max_history = 50                 # visits and searches kept in history

[emoji]
enabled = true

[clock]
enabled = true
format  = "Mon 2 Jan - 15:04"    # Go reference-time layout
zones   = []                     # extra zones: city names ("Tokyo") or IANA ("Asia/Tokyo"), cycle with Ctrl-t

[battery]
enabled = true
device  = "BAT0"                 # name under /sys/class/power_supply

[help]
enabled = true
```

## Data

Calculator, clipboard and web history are small JSON files under
`$XDG_DATA_HOME/launtui/` (usually `~/.local/share/launtui/`), each capped by
its mode's `max_history`. Currency rates are cached for a day under
`$XDG_CACHE_HOME/launtui/`. Delete either directory to start fresh; `Alt-Del`
in a mode clears that mode's history from within launtui.

## Contributing

Contributions are welcome - open an issue or a pull request. You're welcome
to contribute with AI assistance too. Just read and test your code before
submitting.

## License

[MIT](LICENSE) © Raj Nakarja
