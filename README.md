# <img width="150" alt="llmux-icon" src="https://github.com/user-attachments/assets/dc16721e-b884-48d3-851b-1d481cb8c159" /> llmux


A workspace manager for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Run multiple isolated Claude sessions — each workspace gets its own authentication, settings, and history.

## Why

Claude Code stores everything in `~/.claude`. If you work across multiple projects, they all share the same session. llmux gives each project its own `CLAUDE_CONFIG_DIR`, so you get:

- **Isolated sessions** — separate auth, history, and settings per project
- **Automatic routing** — `claude` just works based on your current directory
- **Zero friction** — no manual env vars, no wrapper scripts

## Features

- **Isolated sessions** — each workspace gets its own auth, history, and settings
- **Automatic routing** — `claude` resolves the right workspace based on your current directory
- **Default workspace** — set a fallback workspace for directories without a match
- **Per-workspace API keys** — use different Anthropic API keys per project
- **Worktree mode** — auto-pass `--worktree` to Claude per workspace, bypass with `--no-worktree`
- **Disable attributions** — remove "Made with Claude Code" from commits and PRs per workspace
- **TUI manager** — add, configure, and delete workspaces interactively
- **Shell integration** — supports zsh, bash, and fish

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/alansikora/llmux/main/install.sh | sh
```

Then set up shell integration:

```bash
llmux init zsh    # writes to ~/.zshrc
llmux init bash   # writes to ~/.bashrc
llmux init fish   # writes to ~/.config/fish/config.fish
```

Restart your shell, and you're done.

<details>
<summary>Other install methods</summary>

**With Go:**

```bash
go install github.com/allskar/llmux@latest
```

**From source:**

```bash
git clone https://github.com/alansikora/llmux.git
cd llmux
go build -o llmux .
```

**Custom install directory** (default: `~/.local/bin`)**:**

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/alansikora/llmux/main/install.sh | sudo sh
```

</details>

## Usage

### TUI

```bash
llmux
```

Opens an interactive manager:

| Key | Action |
|-----|--------|
| `a` | Add workspace |
| `o` | Edit workspace options |
| `s` | Toggle default workspace (shown with ★) |
| `d` / `x` | Delete workspace |
| `↑` / `↓` | Navigate |
| `esc` | Return to list |

### Default workspace

Press `s` to set a workspace as the default. When you run `claude` from a directory that doesn't match any workspace, the default is used instead of erroring.

### Workspace options

Press `o` to configure a workspace:

- **Disable attributions** — removes "Made with Claude Code" from commits and PRs
- **Always use worktree** — automatically passes `--worktree` to Claude. Bypass for a single session with `claude --no-worktree`

### Commands

```bash
llmux list              # list all workspaces with auth status
llmux init zsh --print  # print the shell function without installing
```

### How it works

After running `llmux init zsh`, your shell has a thin `claude()` wrapper:

```bash
claude() {
  local config_dir
  config_dir="$(/path/to/llmux resolve "$(pwd -P)")"
  if [ $? -ne 0 ]; then
    echo "llmux: no workspace configured for $(pwd -P)" >&2
    echo "Run 'llmux' to manage workspaces." >&2
    return 1
  fi
  CLAUDE_CONFIG_DIR="$config_dir" command claude "$@"
}
```

When you run `claude` in any directory, the wrapper calls `llmux resolve` to find the matching workspace using longest-prefix path matching. The resolved session directory is passed as `CLAUDE_CONFIG_DIR`.

Workspaces and sessions are stored in `~/.config/llmux/`:

```
~/.config/llmux/
├── config.json              # workspace definitions
└── sessions/
    ├── myapp/               # CLAUDE_CONFIG_DIR for "myapp"
    │   ├── .credentials.json
    │   └── settings.json
    └── backend/
        ├── .credentials.json
        └── settings.json
```

## License

MIT
