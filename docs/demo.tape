# Records docs/demo.gif. Run docs/demo-fixture.sh first, then `vhs docs/demo.tape`
# from the repository root: the fixture holds the invented apps, projects,
# passwords and history the recording shows.

Output docs/demo.gif

Set Shell "bash"
Set FontFamily "RobotoMono Nerd Font Mono"
Set FontSize 20
Set Width 1000
Set Height 540
Set Padding 20
Set Framerate 24
Set TypingSpeed 80ms
Set Theme "Catppuccin Mocha"

Hide
Type "source docs/.demo/env.sh"
Enter
Sleep 1s
Type "launtui"
Enter
Sleep 2s
Show

Sleep 1500ms
Type "fire"
Sleep 2s
Backspace 4
Sleep 500ms

Type "2^10 + 24"
Sleep 2500ms
Backspace 9
Sleep 500ms

Type "100 usd to eur"
Sleep 2500ms
Backspace 14
Sleep 500ms

Type "255 to hex"
Sleep 2500ms
Backspace 10
Sleep 500ms

Ctrl+P
Sleep 2s
Type "git"
Sleep 2s
Backspace 3
Sleep 500ms

Ctrl+O
Sleep 3s

Ctrl+E
Sleep 1500ms
Type "rocket"
Sleep 2500ms
Backspace 6
Sleep 500ms

Ctrl+V
Sleep 3s

Ctrl+H
Sleep 3500ms
Escape
Sleep 1s

Escape
Sleep 700ms
