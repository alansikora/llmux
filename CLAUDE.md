# llmux

Workspace manager for Claude Code. Enables isolated Claude sessions with separate authentication, settings, and history per project.

## Project structure

```
cmd/             # CLI commands (cobra)
internal/
  config/        # Workspace config, path resolution, persistence
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

- **Workspace resolution** uses longest-prefix path matching with path-separator boundaries
- **Config** stored as JSON in `~/.config/llmux/` (overridable via `LLMUX_CONFIG_DIR`)
- **Session data** lives in `~/.config/llmux/sessions/{workspace}/`
- **Shell integration** generates a `claude()` wrapper function that calls `llmux resolve` to route to the correct workspace
- **TUI** is a state machine with 7 states (list, adding, options, sessions, etc.)

## Rules

- **Minimize shell code.** All logic must live in Go. The generated shell wrappers (`internal/shell/generate.go`) and `install.sh` should be kept as thin as possible — just enough to call into the Go binary. Never add new bash/zsh/fish logic when it can be handled in Go instead.
- No automated tests exist yet. Be careful with refactors.
