# llmux

A workspace manager for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Run multiple isolated Claude sessions — each project gets its own authentication, settings, and history.

## Why

Claude Code stores everything in `~/.claude`. If you work across multiple projects, they all share the same session. llmux gives each project its own `CLAUDE_CONFIG_DIR`, so you get:

- **Isolated sessions** — separate auth, history, and settings per project
- **Automatic routing** — `claude` just works based on your current directory
- **Zero friction** — no manual env vars, no wrapper scripts

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

**Custom install directory:**

```bash
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/alansikora/llmux/main/install.sh | sh
```

</details>

## Usage

### TUI

```bash
llmux
```

Opens an interactive manager. Press `a` to add a workspace, `d` to delete one.

When adding a workspace you provide a folder path and a name. llmux also lets you configure defaults like disabling commit/PR attributions.

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
