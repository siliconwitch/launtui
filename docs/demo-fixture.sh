#!/bin/sh
# Builds the throwaway environment docs/demo.tape records against, so the demo
# shows invented data instead of whoever is holding the keyboard.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
demo=$root/docs/.demo

rm -rf "$demo"
mkdir -p "$demo/data/applications" "$demo/data/launtui" "$demo/cache/launtui" \
    "$demo/config/launtui" "$demo/bin" "$demo/projects" "$demo/home"

CGO_ENABLED=0 go build -o "$demo/bin/launtui" "$root"

for app in Blender Calculator Files Firefox GIMP Inkscape Kdenlive KiCad Signal Spotify Steam Thunderbird; do
    lower=$(printf '%s' "$app" | tr 'A-Z' 'a-z')

    cat > "$demo/data/applications/$lower.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=$app
Exec=/bin/true
Terminal=false
EOF
done

cat > "$demo/bin/pass" <<'EOF'
#!/bin/sh
case "$1" in
show)
    printf 'correct-horse-battery-staple\ndemo@example.com\n'
    ;;
*)
    cat <<'TREE'
Password Store
├── email
│   ├── fastmail
│   └── proton
├── social
│   ├── github
│   └── mastodon
├── shopping
│   └── bookshop
├── work
│   ├── gitlab
│   └── vpn
└── router
TREE
    ;;
esac
EOF
chmod +x "$demo/bin/pass"

export GIT_AUTHOR_NAME=Demo GIT_AUTHOR_EMAIL=demo@example.com
export GIT_COMMITTER_NAME=Demo GIT_COMMITTER_EMAIL=demo@example.com

for project in launtui nixos-config portfolio-site sensor-firmware tempo-tracker; do
    mkdir -p "$demo/projects/$project"
    git -C "$demo/projects/$project" init -q -b main
    echo "# $project" > "$demo/projects/$project/README.md"
    git -C "$demo/projects/$project" add -A
    git -C "$demo/projects/$project" commit -qm "Initial commit"
done

echo wip >> "$demo/projects/sensor-firmware/README.md"
echo draft > "$demo/projects/portfolio-site/notes.txt"
git -C "$demo/projects/tempo-tracker" checkout -qb feature/beat-grid

now=$(date +%s)

python3 - "$demo" "$now" <<'EOF'
import json, sys

demo, now = sys.argv[1], int(sys.argv[2])

clips = [
    "https://github.com/siliconwitch/launtui",
    "sha256-4EomlPmKRU1Ptq1oHicZo8MtUBYAqgx6AI6Xy3aG/fo=",
    "The quick brown fox jumps over the lazy dog",
    "docker compose up -d --build",
    "192.168.1.42",
]

web = [
    ("bubbletea docs", "", "bubbletea docs"),
    ("pkg.go.dev", "https://pkg.go.dev", ""),
    ("wayland clipboard protocol", "", "wayland clipboard protocol"),
]

calculations = [("(1920*1080)/2", "1036800"), ("0xff xor 0x0f", "240"), ("sqrt(2)", "1.414214")]

json.dump([{"text": text, "time": now - 60 * (i + 1)} for i, text in enumerate(clips)],
          open(f"{demo}/data/launtui/clipboard-history.json", "w"))

json.dump([{"label": label, "url": url, "query": query, "time": now - 300 * (i + 1)}
           for i, (label, url, query) in enumerate(web)],
          open(f"{demo}/data/launtui/web-history.json", "w"))

json.dump([{"expression": expression, "answer": answer, "time": now - 120 * (i + 1)}
           for i, (expression, answer) in enumerate(calculations)],
          open(f"{demo}/data/launtui/calculator-history.json", "w"))

json.dump({"fetched": now,
           "rates": {"EUR": 1.0, "USD": 1.09, "GBP": 0.84, "JPY": 171.2, "SEK": 11.02, "CHF": 0.93}},
          open(f"{demo}/cache/launtui/currency-rates.json", "w"))
EOF

cat > "$demo/config/launtui/config.toml" <<EOF
[projects]
dirs = ["$demo/projects"]
EOF

cat > "$demo/env.sh" <<EOF
export XDG_DATA_HOME=$demo/data
export XDG_CACHE_HOME=$demo/cache
export XDG_CONFIG_HOME=$demo/config
export XDG_DATA_DIRS=$demo/data
export PATH=$demo/bin:\$PATH
export HOME=$demo/home
export PS1='\$ '
clear
EOF
