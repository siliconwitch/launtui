# CLAUDE.md

Guidance for working in this repository. This file holds **only** high-level
coding and operational principles plus the architectural roles below. It must
never describe product functionality or per-feature behaviour. Keep it that way.

## Coding principles

General and meant to be reused verbatim across projects.

- **Complete names.** Use descriptive, whole-word names for non-trivial
  variables (`tuiWidth`, not `boxW`). Short names are acceptable only for
  receivers, loop indices, `err`, and `ok`.
- **Breathing room.** Separate a statement that produces a value from the
  statement that consumes it with a blank line — e.g. an assignment, a blank
  line, then the `if err != nil` check. Group code into readable paragraphs.
- **Guard clauses.** Handle edge cases and errors first and return early, so the
  happy path stays unindented and reads straight down the function.
- **No comments.** Code must explain itself through naming and structure.
  (Struct tags are not comments.)
- **Procedural code.** Always inline simple logic so readers don't have to jump
  around to see what small functions do. 1-3 line functions shouldn't exist
  unless there's a very good reason — e.g. they wrap something that could change,
  such as a hardcoded filepath. Never create a function that is only used once.
  Long procedural functions are fine; reading one top to bottom should describe
  its entire behaviour with minimal jumping around.
- **Functional code.** Prefer functional, stateless code. Some libraries demand
  statefulness and it's fine to follow their style, but everywhere else avoid
  mutable state.
- **Switch over ladders.** Prefer a `switch` (including a type switch) to a long
  `if` / `else if` chain.
- **Co-location.** A self-contained unit lives entirely in its own file — its
  config, state, behaviour, and rendering together. Its only references from
  elsewhere are where it is wired in at the composition root. Removing it means
  deleting its file and that one line of wiring — nothing scattered across the
  project.
- **Table-driven tests.** Express tests as a table of input → expected cases
  iterated in a loop, not as repeated near-identical assertions.

## Architecture principles

- **The Elm Architecture (Bubble Tea).** State lives in models, transitions
  happen in `Update`, side effects are expressed as `Cmd`s. Never block and
  never spawn goroutines directly — express asynchronous work as a `Cmd`.
- **Widgets are independent.** They coordinate only through messages, never by
  calling one another.
- **No import cycles.** Widgets never import the `tui` package; they satisfy the
  loader's interface structurally.
- **Capability interfaces, not fat ones.** A unit implements a small core
  interface and opts into extra behaviour only by satisfying additional, single-
  purpose interfaces that the composition root detects with a type assertion
  (`Mode`, plus optional `StrongMatcher`, `RowDeleter`, `Selectable`). New
  capabilities never widen the core interface.
- **Declarative, decentralized config.** Each unit owns its config struct,
  defaults, and section name; one generic, unit-agnostic loader overlays the
  on-disk file. Adding or changing config never touches the loader.

## Operational principles

- Build and run with cgo disabled for a static, dependency-free binary:
  `CGO_ENABLED=0 go build`.

## Roles

- **`main.go`** — process entry point. Parses flags, constructs the
  application (or dispatches to a widget-provided auxiliary process mode),
  runs the program, and reports startup and config errors. No feature logic.
- **`internal/tui/app.go`** — the root model. Owns the widgets, routes incoming
  messages to them, holds global state (window size and the selection cursor),
  and composes their rendered output into the overall layout. The single place
  widgets are wired together. Also owns the shared search input and the set of
  modes: it tracks the current mode, switches automatically to the first mode
  that has results for the query (unless a hotkey or startup flag has pinned
  one), and feeds the query to every mode. It owns the one selection cursor for
  the active mode — moving, clamping, and resetting it on a query change or mode
  switch — and draws the shared results box from that mode's rows. Also holds
  the generic configuration loader: it
  resolves the config path (`$LAUNTUI_CONFIG` overrides the default, which
  also gives tests a hermetic seam) and overlays the on-disk file onto each
  widget's defaults; the loader is widget-agnostic and never changes when
  widgets are added or removed.

## Widget structure

Every widget lives in `internal/widgets/<name>.go` and contains, together: its
config struct (with `toml` tags) and defaults, the method exposing its config
section name, its model type and constructor, its message handling, and its
rendering helpers, plus any private message types it needs. Shared,
widget-agnostic code lives in `internal/widgets/widgets.go`: the mode
interfaces and app-level messages, the `Row` descriptor and the shared results
renderer that draws an active mode's rows with cursor, windowing, and
right-aligned columns, the fuzzy-filtered `list` that stores and filters a
mode's items, shared styles, text truncation, and the history slice helpers,
together with the OS-integration facilities used across widgets — clipboard
access, the shared clipboard-history store and its suppression protocol (a
copy-recording sink written by several widgets and a background process, owned
by none), JSON storage under the XDG directories, home-path expansion, and
launching processes (detached or wrapped in a terminal emulator). These are
shared facilities, not any widget's behaviour: a widget that records a copy or
launches a process calls them rather than reaching into another widget.
A mode whose UI is a single filterable list holds a `list` field and maps its
filtered items to rows in `Rows()`, keeping only its own activation logic.

A widget that can be hidden carries an `Enabled bool` (toml `enabled`, default
`true`) in its config and exposes an `Enabled() bool` method; each widget
guards its own startup `Cmd` with it, and `app.go` consults it to omit the
widget from the layout.

A **mode** is a searchable widget (Run, Calculator, …) that satisfies the
`widgets.Mode` interface in `widgets.go`: it takes the shared query and exposes
its current `Rows()` (each row carrying its left/right content and its dim and
deletable flags), its accent colour, an activate-by-index, and an optional
status line shown above the rows. The app owns the cursor and draws the rows,
so modes hold no navigation or windowing code. A mode "has results" — and so
claims the query during auto-switching — when it produces at least one non-dim
row. Modes share the app-owned input rather than carrying their own, and
declare their display name and `ctrl`-hotkey so the mode bar, auto-switching,
startup flags (`-<letter>` maps to `ctrl+<letter>`), and help stay in sync.
Adding a mode means writing its file and listing it once in `app.go`; their
order there is the auto-switch priority — selective matchers first, the
catch-all mode last. A mode may additionally satisfy `widgets.StrongMatcher` to
claim a query ahead of the normal order when it recognises the query with high
confidence. A mode whose rows can be deleted additionally satisfies
`widgets.RowDeleter`; `app.go` routes the delete (the selected deletable row)
and alt+delete (clear all) keys through it. A mode that resolves extra row
detail lazily for the highlighted entry satisfies `widgets.Selectable`;
`app.go` calls its `Select(index)` whenever the selection moves.

Quitting is owned by `app.go`: widget `Cmd`s never return `tea.QuitMsg`
(bubbletea short-circuits it before `Update`); they return
`widgets.RequestQuitMsg` instead. `app.go` answers it (and `esc`) by
broadcasting `widgets.AppClosingMsg` to every mode, so widgets can persist
state in a final `Cmd` before the app quits.

## Maintenance

After adding or changing a major feature, re-read this file and update it so the
principles, roles, and structure stay accurate.
