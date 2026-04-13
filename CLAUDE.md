# llmux

Session manager for Claude Code. Enables isolated Claude sessions with separate authentication, settings, and history per project, grouped into reusable auth profiles.

## Project structure

```
cmd/             # CLI commands (cobra)
internal/
  config/        # Profile/project config, path resolution, persistence
  tui/           # Interactive terminal UI (bubbletea + huh + lipgloss)
  shell/         # Shell integration generator (claude() wrapper)
  worktree/      # Git worktree session management
main.go          # Entry point
install.sh       # Download & install script
.goreleaser.yml  # Release config
```

## Build & run

```sh
go build -o llmux .
go run .
```

Version is set via ldflags: `-X main.version=v{version}`

## Key dependencies

- `spf13/cobra` — CLI framework
- `charmbracelet/bubbletea` — TUI framework
- `charmbracelet/huh` — TUI forms
- `charmbracelet/lipgloss` — TUI styling

## Architecture notes

- **Profile** = auth/credentials bundle (OAuth subscription or API key). Session directories with `.claude.json` and `settings.json` are per-profile.
- **Project** = a directory path + profile reference. Projects are the primary entity — the main TUI view is a flat alphabetical list of all projects.
- **Profile resolution** uses longest-prefix path matching against registered project paths (with path-separator boundaries). If no project matches, falls back to the default profile.
- **Config** stored as JSON in `~/.config/llmux/` (overridable via `LLMUX_CONFIG_DIR`). Legacy configs with `workspaces` are auto-migrated to `profiles` on load.
- **Session data** lives in `~/.config/llmux/sessions/{profile}/`
- **Shell integration** generates a `claude()` wrapper function that calls `llmux resolve` to route to the correct profile based on `pwd`.
- **TUI** is a state machine with 10 states (project list as main view, plus project/profile/session forms and views).

## Rules

- **Minimize shell code.** All logic must live in Go. The generated shell wrappers (`internal/shell/generate.go`) and `install.sh` should be kept as thin as possible — just enough to call into the Go binary. Never add new bash/zsh/fish logic when it can be handled in Go instead.
- No automated tests exist yet. Be careful with refactors.
