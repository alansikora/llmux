# Usage reference

Full keybindings and command details for llmux. See the [README](README.md) for installation and overview.

## TUI keybindings

### Project list (main view)

A flat, alphabetical list of every registered project.

| Key | Action |
|-----|--------|
| `enter` | View worktree sessions for the selected project |
| `a` | Add project (pick folder + profile) |
| `e` | Edit project overrides |
| `d` / `x` | Remove project (inline, no confirmation) |
| `p` | Manage profiles |
| `o` | General options |
| `esc` / `ctrl+c` | Quit |

### Profile list

Press `p` from the project list to manage profiles (auth/credentials bundles).

| Key | Action |
|-----|--------|
| `enter` / `e` | Edit profile options |
| `a` | Add profile |
| `r` | Rename profile |
| `s` | Toggle default profile (shown with ★) |
| `d` / `x` | Delete profile (confirmation required) |
| `esc` | Back to projects |

### Session list

| Key | Action |
|-----|--------|
| `a` / `enter` | Apply session |
| `u` | Unapply current session |
| `c` | Copy worktree path to clipboard |
| `d` / `x` | Delete session (confirmation required — destroys the git worktree) |
| `esc` | Back to project list |

## Profile options

Press `e` on a profile to configure defaults for all projects using it:

- **Disable attributions** — removes "Made with Claude Code" from commits and PRs
- **Always use worktree** — automatically passes `--worktree` to Claude

### Project overrides

Press `e` on a project to override its profile's defaults. Each setting can be **inherit** (use profile default), **enabled**, or **disabled**.

Available overrides:
- Worktree mode
- Disable attributions

## General options

Press `o` from the project list to configure global settings:

- **Short alias** — defines `c` as a shorthand for `claude`
- **Apply marker** — creates a `.llmux-applied` file when a session is applied
- **Auto mode** — passes `--enable-auto-mode` to Claude Code
- **Status line** — enables [ccstatusline](https://github.com/sirmalloc/ccstatusline) across all profiles

## Worktree session workflow

### Applying a session

```bash
llmux apply [session]       # from your main working tree
llmux apply -p <profile>    # specify profile explicitly
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

Finds the worktree, sets up the profile environment, and launches `claude --continue` inside it. No need to manually `cd` or pass `--no-worktree`.

## Slash commands

Installed automatically during `llmux init` into `~/.claude/commands/llmux/`:

- `/llmux apply` — apply the current worktree session's changes to the main working tree
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
