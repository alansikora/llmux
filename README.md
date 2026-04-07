# <img width="150" alt="llmux-icon" src="https://github.com/user-attachments/assets/dc16721e-b884-48d3-851b-1d481cb8c159" /> llmux


A workspace manager for AI coding assistants. Currently supports [Claude Code](https://docs.anthropic.com/en/docs/claude-code), with more CLIs planned. Run multiple isolated sessions — each workspace gets its own authentication, settings, and history.

## Why

Claude Code stores everything in `~/.claude`. If you work across multiple projects, they all share the same session. llmux gives each project its own `CLAUDE_CONFIG_DIR`, so you get:

- **Isolated sessions** — separate auth, history, and settings per project
- **Automatic routing** — `claude` just works based on your current directory
- **Zero friction** — no manual env vars, no wrapper scripts

## Features

- **Workspaces & projects** — workspaces hold auth and default settings; projects map directories to workspaces with optional per-project overrides
- **Automatic routing** — `claude` resolves the right workspace based on your current directory
- **Auto-registration** — run `claude` in an unregistered directory and get prompted to pick a workspace
- **Default workspace** — set a fallback workspace for directories without a match
- **Per-workspace API keys** — use different Anthropic API keys per project
- **Worktree mode** — auto-pass `--worktree` to Claude per workspace or project, bypass with `--no-worktree` / `-nw`
- **Worktree session management** — list, apply, revert, and resume Claude worktree sessions
- **Session resume** — resume a worktree session by name or branch with `llmux resume`
- **Slash commands** — `/llmux apply` and `/llmux unapply` available inside Claude Code sessions
- **Auto mode** — globally pass `--enable-auto-mode` to Claude Code
- **Status line** — enable [ccstatusline](https://github.com/sirmalloc/ccstatusline) across all workspaces, showing model, context, and git info in Claude Code
- **Disable attributions** — remove "Made with Claude Code" from commits and PRs per workspace
- **Auto-update** — `llmux upgrade` checks GitHub releases and upgrades in place, with update notices in the TUI
- **Short alias** — optionally define `c` as a shorthand for `claude`
- **TUI manager** — add, configure, and manage workspaces, projects, and sessions interactively
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

**Canary (latest from `main`):**

```bash
curl -fsSL https://raw.githubusercontent.com/alansikora/llmux/main/install.sh | sh -s -- --canary
```

</details>

## Usage

### TUI

```bash
llmux
```

Opens an interactive manager. The TUI has three levels: workspaces → projects → sessions.

**Workspace list:**

| Key | Action |
|-----|--------|
| `enter` | View projects |
| `a` | Add workspace |
| `e` | Edit workspace options |
| `o` | General options |
| `s` | Toggle default workspace (shown with ★) |
| `d` / `x` | Delete workspace |
| `esc` | Quit |

**Project list** (inside a workspace):

| Key | Action |
|-----|--------|
| `enter` | View worktree sessions |
| `a` | Add project |
| `e` | Edit project overrides |
| `d` / `x` | Remove project |
| `esc` | Back to workspaces |

### Default workspace

Press `s` to set a workspace as the default. When you run `claude` from a directory that doesn't match any workspace, the default is used instead of erroring.

### Workspace options

Press `e` on a workspace to configure defaults that apply to all its projects:

- **Disable attributions** — removes "Made with Claude Code" from commits and PRs
- **Always use worktree** — automatically passes `--worktree` to Claude. Bypass for a single session with `claude --no-worktree` (or `-nw`)

### Project overrides

Press `e` on a project to override workspace defaults for that directory. Settings can be set to **inherit** (use workspace default), **enabled**, or **disabled**.

### Worktree sessions

When Claude Code runs with `--worktree`, it creates a git worktree under `.claude/worktrees/` with changes on a separate branch. Use these commands to manage those sessions:

```bash
llmux sessions              # list worktree sessions for the current project
llmux resume <name>         # resume a session by name or branch (launches claude --continue)
llmux apply [session]       # apply session changes as uncommitted diffs on main
llmux unapply               # revert applied changes (restores any auto-stashed state)
```

`llmux resume` finds the worktree, sets up the workspace environment, and launches Claude directly inside it — no need to manually `cd` or pass `--no-worktree`.

`llmux apply` auto-detects the session name when run from inside a worktree. Changes are applied as uncommitted modifications — no merge commits. If your working tree is dirty, llmux auto-stashes first and restores on `unapply`.

You can also browse and manage sessions from the TUI (workspace → project → `enter`):

| Key | Action |
|-----|--------|
| `a` / `enter` | Apply session |
| `u` | Unapply current session |
| `c` | Copy worktree path to clipboard |
| `d` | Delete session |
| `esc` | Back to project list |

### Slash commands

llmux installs Claude Code slash commands during `llmux init`:

- `/llmux apply` — apply the current worktree session's changes to the main workspace
- `/llmux unapply` — revert previously applied changes

These work inside any Claude Code session, including worktree sessions. The shell wrapper automatically skips adding `--worktree` when you use `--resume` or `--continue`, so resuming existing sessions works without needing `--no-worktree`.

### General options

Press `o` to configure global settings:

- **Short alias** — defines `c` as a shorthand for `claude` (requires shell restart to take effect)
- **Apply marker** — creates a `.llmux-applied` file in the workspace root when a session is applied (makes it visible in `git status`)
- **Auto mode** — passes `--enable-auto-mode` to Claude Code, allowing it to run without confirmation prompts
- **Status line** — enables [ccstatusline](https://github.com/sirmalloc/ccstatusline) across all workspaces. When enabled, the status line config is automatically added to every workspace's `settings.json` (including newly created workspaces). Disable to remove from all workspaces.

### CLI commands

```bash
llmux                   # open the TUI manager
llmux list              # list all workspaces and projects with auth status
llmux sessions          # list worktree sessions for the current project
llmux resume <name>     # resume a worktree session by name or branch
llmux apply [session]   # apply worktree session changes to main
llmux unapply           # revert applied changes
llmux upgrade           # upgrade to the latest version
llmux init zsh --print  # print the shell function without installing
```

### How it works

After running `llmux init zsh`, your shell has a thin `claude()` wrapper that calls `llmux resolve` when you run `claude` in any directory. The resolve command finds the matching workspace using longest-prefix path matching against registered projects, and returns the session directory, API key, worktree flag, and auto-mode flag.

If the directory isn't registered to any workspace, the wrapper triggers `llmux register` to let you interactively pick a workspace.

If the resolved project has **Always use worktree** enabled, the wrapper also runs `git fetch origin <default-branch>` before launching Claude — so the worktree is always based on an up-to-date branch. The fetch is a best-effort no-op if there's no remote, no network, or `origin/HEAD` isn't set. Pass `--no-worktree` (or `-nw`) to skip both the fetch and the worktree for a single session.

Workspaces and sessions are stored in `~/.config/llmux/` (overridable via `LLMUX_CONFIG_DIR`):

```
~/.config/llmux/
├── config.json              # workspace + project definitions
└── sessions/
    ├── myapp/               # CLAUDE_CONFIG_DIR for "myapp" workspace
    │   ├── .credentials.json
    │   └── settings.json
    └── backend/
        ├── .credentials.json
        └── settings.json
```

## License

MIT
