# <img width="150" alt="llmux-icon" src="https://github.com/user-attachments/assets/dc16721e-b884-48d3-851b-1d481cb8c159" /> llmux

[![GitHub release](https://img.shields.io/github/v/release/alansikora/llmux)](https://github.com/alansikora/llmux/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Workspace manager for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Run multiple isolated sessions — each project gets its own authentication, settings, and history. No manual env vars, no wrapper scripts.

<!-- TODO: add a GIF/screenshot of the TUI here -->

## Quick start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/alansikora/llmux/main/install.sh | sh

# Set up your shell
llmux init zsh   # or bash / fish

# Restart your shell, then launch the TUI
llmux
```

From the TUI: create a workspace, add a project directory, and you're done. Run `claude` anywhere and llmux handles the rest.

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

## Why

Claude Code stores everything in `~/.claude`. If you work across multiple projects — personal repos, work repos, client projects — they all share the same session, API key, and settings. llmux gives each project its own `CLAUDE_CONFIG_DIR`, so you get full isolation with zero friction.

## Features

### Workspaces and routing

- **Isolated sessions** — each workspace gets its own auth, history, settings, and API key
- **Automatic routing** — a thin shell wrapper resolves the right workspace based on your current directory. Register a parent directory and all subdirectories inherit it; register a more specific child path and it gets its own workspace while siblings keep the parent's
- **Auto-registration** — run `claude` in an unregistered directory and get prompted to pick a workspace
- **Default workspace** — set a fallback for directories that don't match any project
- **Subcommand passthrough** — `claude mcp`, `claude config`, etc. bypass workspace resolution and work natively

### Worktree sessions

When a workspace has **Always use worktree** enabled, Claude Code runs in an isolated git worktree under `.claude/worktrees/`. This keeps your main branch clean while Claude works on a separate branch.

- **Apply and unapply** — bring session changes into your main workspace as uncommitted diffs, then cleanly revert them
- **Full working-state capture** — apply picks up committed, staged, _and_ unstaged changes (including untracked files)
- **3-way merge** — conflicts during apply are handled gracefully via `git apply --3way`
- **Safe with dirty trees** — if your working tree has uncommitted changes, llmux auto-stashes before apply and restores on unapply
- **Diff-based unapply** — reversal uses a saved diff, so edits you make after applying are preserved
- **Stale detection** — sessions older than 7 days are marked as stale in the TUI
- **Resume** — resume any session by name or branch without manually navigating to the worktree
- **Auto-fetch** — worktree mode fetches `origin/<default-branch>` before launching Claude, so worktrees start from an up-to-date base
- **Slash commands** — `/llmux apply` and `/llmux unapply` work directly inside Claude Code sessions

### Customization

- **Per-workspace API keys** — use different Anthropic API keys per project
- **Disable attributions** — remove "Made with Claude Code" from commits and PRs (per workspace or per project)
- **Auto mode** — globally pass `--enable-auto-mode` to Claude Code
- **Status line** — enable [ccstatusline](https://github.com/sirmalloc/ccstatusline) showing model, context, and git info
- **Apply marker** — drop a `.llmux-applied` file when a session is applied (visible in `git status`)
- **Short alias** — optionally define `c` as a shorthand for `claude`
- **Project overrides** — override any workspace default (worktree, attributions) per project directory

## Usage

### TUI

```bash
llmux
```

Opens an interactive manager with three levels: **workspaces → projects → sessions**. Press `?` or see the on-screen help for keybindings.

See [USAGE.md](USAGE.md) for the full keybindings reference.

### CLI

```bash
llmux                       # open the TUI manager
llmux list                  # list all workspaces and projects with auth status
llmux sessions [workspace]  # list worktree sessions for the current project
llmux resume <name>         # resume a session by name or branch
llmux apply [session]       # apply session changes to main (-w to specify workspace)
llmux unapply               # revert applied changes
llmux upgrade               # upgrade to the latest version
llmux init <shell> [--print]  # set up shell integration (or print without installing)
```

`llmux apply` auto-detects the session when run from inside a worktree. Pass `--no-worktree` (or `-nw`) to skip worktree mode for a single `claude` invocation. `--resume`, `--continue`, `-r`, and `-c` also bypass worktree injection automatically.

### How it works

After `llmux init`, your shell has a thin `claude()` wrapper. When you run `claude`, it calls `llmux resolve` to find the matching workspace using longest-prefix path matching, then launches Claude Code with the right `CLAUDE_CONFIG_DIR`, API key, and flags.

Claude subcommands (like `claude mcp add`) are detected and passed through directly — no workspace resolution overhead. The subcommand list is cached locally with a 24-hour TTL.

Config is stored in `~/.config/llmux/` (override with `LLMUX_CONFIG_DIR`):

```
~/.config/llmux/
├── config.json              # workspace + project definitions
└── sessions/
    ├── myapp/               # CLAUDE_CONFIG_DIR for "myapp"
    │   ├── .credentials.json
    │   └── settings.json
    └── backend/
        ├── .credentials.json
        └── settings.json
```

## Shell support

Supports **zsh**, **bash**, and **fish**. The generated wrapper is intentionally minimal — all logic lives in the Go binary.

## License

MIT
