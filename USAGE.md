# Usage reference

Full keybindings and command details for llmux. See the [README](README.md) for installation and overview.

## TUI keybindings

### Workspace list

| Key | Action |
|-----|--------|
| `enter` | View projects |
| `a` | Add workspace |
| `r` | Rename workspace |
| `e` | Edit workspace options |
| `o` | General options |
| `s` | Toggle default workspace (shown with ★) |
| `d` / `x` | Delete workspace |
| `esc` | Quit |

### Project list

| Key | Action |
|-----|--------|
| `enter` | View worktree sessions |
| `a` | Add project |
| `e` | Edit project overrides |
| `d` / `x` | Remove project |
| `esc` | Back to workspaces |

### Session list

| Key | Action |
|-----|--------|
| `a` / `enter` | Apply session |
| `u` | Unapply current session |
| `c` | Copy worktree path to clipboard |
| `d` | Delete session |
| `esc` | Back to project list |

## Workspace options

Press `e` on a workspace to configure defaults for all its projects:

- **Disable attributions** — removes "Made with Claude Code" from commits and PRs
- **Always use worktree** — automatically passes `--worktree` to Claude

### Project overrides

Press `e` on a project to override workspace defaults. Each setting can be **inherit** (use workspace default), **enabled**, or **disabled**.

Available overrides:
- Worktree mode
- Disable attributions

## General options

Press `o` from the workspace list to configure global settings:

- **Short alias** — defines `c` as a shorthand for `claude`
- **Apply marker** — creates a `.llmux-applied` file when a session is applied
- **Auto mode** — passes `--enable-auto-mode` to Claude Code
- **Status line** — enables [ccstatusline](https://github.com/sirmalloc/ccstatusline) across all workspaces

## Worktree session workflow

### Applying a session

```bash
llmux apply [session]       # from your main workspace
llmux apply -w <workspace>  # specify workspace explicitly
```

When run from inside a worktree, the session name is auto-detected.

**What happens during apply:**
1. If your working tree is dirty, llmux auto-stashes your changes
2. The session's full state (committed + staged + unstaged + untracked) is captured
3. A diff from your main HEAD to the session snapshot is generated
4. The diff is applied with `git apply --3way` for graceful conflict handling
5. The diff is saved for clean reversal later

### Reverting applied changes

```bash
llmux unapply
```

**What happens during unapply:**
1. The saved diff is reverse-applied
2. Any auto-stashed changes are restored
3. The apply marker file is removed (if enabled)

Edits you made after applying are preserved — only the original applied diff is reversed.

### Resuming a session

```bash
llmux resume <name-or-branch>
```

Finds the worktree, sets up the workspace environment, and launches `claude --continue` inside it. No need to manually `cd` or pass `--no-worktree`.

## Slash commands

Installed automatically during `llmux init` into `~/.claude/commands/llmux/`:

- `/llmux apply` — apply the current worktree session's changes to the main workspace
- `/llmux unapply` — revert previously applied changes

These work inside any Claude Code session, including worktree sessions.

## Shell wrapper flags

The `claude()` wrapper respects these flags:

| Flag | Effect |
|------|--------|
| `--no-worktree`, `-nw` | Skip worktree mode for this invocation |
| `--resume`, `-r` | Resume mode — skips worktree injection |
| `--continue`, `-c` | Continue mode — skips worktree injection |

## Environment variables

| Variable | Description |
|----------|-------------|
| `LLMUX_CONFIG_DIR` | Override the config directory (default: `~/.config/llmux`) |
| `INSTALL_DIR` | Custom binary location during install (default: `~/.local/bin`) |
