package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func rcFile(sh string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch sh {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: zsh, bash, fish)", sh)
	}
}

func snippet(bin string) string {
	return fmt.Sprintf(`claude() {
  local resolve_output config_dir api_key worktree_flag
  resolve_output="$(%s resolve "$(pwd -P)")"
  if [ $? -ne 0 ]; then
    echo "llmux: no workspace configured for $(pwd -P)" >&2
    echo "Run 'llmux' to manage workspaces." >&2
    return 1
  fi
  config_dir="$(echo "$resolve_output" | head -n1)"
  api_key="$(echo "$resolve_output" | sed -n '2p')"
  worktree_flag="$(echo "$resolve_output" | sed -n '3p')"
  local args=("$@")
  if [ "$worktree_flag" = "--worktree" ]; then
    local no_worktree=false filtered=()
    for arg in "${args[@]}"; do
      if [ "$arg" = "--no-worktree" ]; then
        no_worktree=true
      else
        filtered+=("$arg")
      fi
    done
    if [ "$no_worktree" = false ]; then
      local default_branch
      default_branch="$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')"
      if [ -n "$default_branch" ]; then
        git fetch origin "$default_branch" 2>/dev/null
      fi
      args=("--worktree" "${filtered[@]}")
    else
      args=("${filtered[@]}")
    fi
  fi
  if [ -n "$api_key" ]; then
    ANTHROPIC_API_KEY="$api_key" CLAUDE_CONFIG_DIR="$config_dir" command claude "${args[@]}"
  else
    CLAUDE_CONFIG_DIR="$config_dir" command claude "${args[@]}"
  fi
}`, bin)
}

func fishSnippet(bin string) string {
	return fmt.Sprintf(`function claude
  set -l resolve_output (string split \n (%s resolve (pwd -P)))
  if test $status -ne 0
    echo "llmux: no workspace configured for "(pwd -P) >&2
    echo "Run 'llmux' to manage workspaces." >&2
    return 1
  end
  set -l config_dir $resolve_output[1]
  set -l api_key ""
  set -l worktree_flag ""
  if test (count $resolve_output) -ge 2
    set api_key $resolve_output[2]
  end
  if test (count $resolve_output) -ge 3
    set worktree_flag $resolve_output[3]
  end
  set -l args $argv
  if test "$worktree_flag" = "--worktree"
    set -l filtered
    set -l no_worktree false
    for arg in $args
      if test "$arg" = "--no-worktree"
        set no_worktree true
      else
        set -a filtered $arg
      end
    end
    if test "$no_worktree" = false
      set -l default_branch (git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')
      if test -n "$default_branch"
        git fetch origin $default_branch 2>/dev/null
      end
      set args --worktree $filtered
    else
      set args $filtered
    end
  end
  if test -n "$api_key"
    ANTHROPIC_API_KEY=$api_key CLAUDE_CONFIG_DIR=$config_dir command claude $args
  else
    CLAUDE_CONFIG_DIR=$config_dir command claude $args
  end
end`, bin)
}

const marker = "# llmux shell integration"

func evalLine(bin, sh string) string {
	return fmt.Sprintf("%s\neval \"$(%s init %s --print)\"", marker, bin, sh)
}

func fishEvalLine(bin, sh string) string {
	return fmt.Sprintf("%s\n%s init %s --print | source", marker, bin, sh)
}

// Generate returns the shell function using the absolute path to the binary.
// If shortAlias is true, an additional "c" alias/function is appended.
func Generate(bin, sh string, shortAlias bool) (string, error) {
	var out string
	switch sh {
	case "zsh", "bash":
		out = snippet(bin)
		if shortAlias {
			out += "\nc() { claude \"$@\"; }"
		}
	case "fish":
		out = fishSnippet(bin)
		if shortAlias {
			out += "\nfunction c; claude $argv; end"
		}
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: zsh, bash, fish)", sh)
	}
	return out, nil
}

// Install appends the eval line to the shell's rc file.
// Returns the path of the modified file.
func Install(bin, sh string) (string, error) {
	rc, err := rcFile(sh)
	if err != nil {
		return "", err
	}

	// Read existing content to check if already installed
	data, err := os.ReadFile(rc)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	content := string(data)
	if strings.Contains(content, marker) {
		return rc, fmt.Errorf("already installed in %s", rc)
	}

	var line string
	switch sh {
	case "fish":
		line = fishEvalLine(bin, sh)
	default:
		line = evalLine(bin, sh)
	}

	// Ensure trailing newline before appending
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		line = "\n" + line
	}

	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, line); err != nil {
		return "", err
	}

	return rc, nil
}
