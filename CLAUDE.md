# CLAUDE.md

Guidance for working in this repository. This file holds **only** high-level
coding and operational principles plus the architectural roles below. It must
never describe product functionality or per-feature behaviour. Keep it that way.

## Coding principles

- **Complete names.** Use descriptive, whole-word names for non-trivial
  variables (`tuiWidth`, not `boxW`). Short names are acceptable only for
  receivers, loop indices, `err`, and `ok`.
- **Breathing room.** Separate a statement that produces a value from the
  statement that consumes it with a blank line — e.g. an assignment, a blank
  line, then the `if err != nil` check. Group code into readable paragraphs.
- **No comments.** Code must explain itself through naming and structure.
  (Struct tags are not comments.)
- **Co-location.** A widget is entirely self-contained in its own file: its
  config, model, behaviour, and rendering live together. No widget-specific
  logic lives anywhere else. The only references to a widget outside its file
  are its wiring in `app.go`. Removing a widget means deleting its file and that
  wiring — nothing is scattered across the project.

## Architecture principles

- **The Elm Architecture (Bubble Tea).** State lives in models, transitions
  happen in `Update`, side effects are expressed as `Cmd`s. Never block and
  never spawn goroutines directly — express asynchronous work as a `Cmd`.
- **Widgets are independent.** They coordinate only through messages, never by
  calling one another.
- **No import cycles.** Widgets never import the `tui` package; they satisfy the
  loader's interface structurally.

## Operational principles

- Build and run with cgo disabled for a static, dependency-free binary:
  `CGO_ENABLED=0 go build`.

## Roles

- **`main.go`** — process entry point. Constructs the application, runs the
  program, and reports startup and config errors. No feature logic.
- **`internal/tui/app.go`** — the root model. Owns the widgets, routes incoming
  messages to them, holds global state (window size), and composes their
  rendered output into the overall layout. The single place widgets are wired
  together. Also owns the shared search input and the set of modes: it tracks
  the current mode, switches automatically to the first mode that has results
  for the query (unless a hotkey or startup flag has pinned one), and feeds the
  query to every mode.
- **`internal/tui/config.go`** — the generic configuration loader. Resolves the
  config path and overlays the on-disk file onto each widget's defaults. It is
  widget-agnostic and never changes when widgets are added or removed.

## Widget structure

Every widget lives in `internal/widgets/<name>.go` and contains, together: its
config struct (with `toml` tags) and defaults, the method exposing its config
section name, its model type and constructor, its message handling, and its
rendering helpers, plus any private message types it needs. Shared,
widget-agnostic styling and helpers live in `internal/widgets/styles.go`.

A widget that can be hidden carries an `Enabled bool` (toml `enabled`, default
`true`) in its config and exposes an `Enabled() bool` method; `app.go` consults
it to skip the widget's startup `Cmd` and omit it from the layout.

A **mode** is a searchable widget (Run, Calculator, …) that satisfies the
`widgets.Mode` interface in `mode.go`: it takes the shared query, reports whether
it has results, navigates and activates a selection, and renders its own list.
Modes share the app-owned input rather than carrying their own, and declare
their display name and `ctrl`-hotkey so the mode bar, auto-switching, and help
stay in sync. Adding a mode means writing its file and listing it once in
`app.go`.

## Maintenance

After adding or changing a major feature, re-read this file and update it so the
principles, roles, and structure stay accurate.
