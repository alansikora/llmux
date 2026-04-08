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
  # Pass-through for non-interactive subcommands — bypass workspace resolution
  # entirely so these work even outside a configured workspace.
  # The subcommand list is maintained in Go (llmux is-subcommand).
  local _llmux_first_pos=""
  for arg in "$@"; do
    [[ "$arg" == -* ]] && continue
    _llmux_first_pos="$arg"
    break
  done
  if [ -n "$_llmux_first_pos" ] && %s is-subcommand "$_llmux_first_pos" 2>/dev/null; then
    local _llmux_pt_output
    _llmux_pt_output="$(%s resolve "$(pwd -P)" -- "$@" 2>/dev/null)"
    if [ $? -eq 0 ]; then
      CLAUDE_CONFIG_DIR="$(echo "$_llmux_pt_output" | head -n1)" command claude "$@"
    else
      command claude "$@"
    fi
    return
  fi

  local resolve_output config_dir extra_flags
  resolve_output="$(%s resolve "$(pwd -P)" -- "$@")"
  local resolve_status=$?
  if [ $resolve_status -eq 2 ]; then
    %s register "$(pwd -P)" || return 1
    resolve_output="$(%s resolve "$(pwd -P)" -- "$@")" || return 1
  elif [ $resolve_status -ne 0 ]; then
    echo "llmux: no workspace configured for $(pwd -P)" >&2
    echo "Run 'llmux' to manage workspaces." >&2
    return 1
  fi
  config_dir="$(echo "$resolve_output" | head -n1)"
  extra_flags="$(echo "$resolve_output" | sed -n '2p')"
  local args=()
  for arg in "$@"; do
    if [ "$arg" = "--no-worktree" ] || [ "$arg" = "-nw" ]; then
      continue
    fi
    args+=("$arg")
  done
  if [ "$extra_flags" = "--worktree" ]; then
    local default_branch
    default_branch="$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')"
    if [ -n "$default_branch" ]; then
      git fetch origin "$default_branch" 2>/dev/null
    fi
    args=("--worktree" "${args[@]}")
  fi
  CLAUDE_CONFIG_DIR="$config_dir" command claude "${args[@]}"
}`, bin, bin, bin, bin, bin)
}

func fishSnippet(bin string) string {
	return fmt.Sprintf(`function claude
  # Pass-through for non-interactive subcommands — bypass workspace resolution
  # entirely so these work even outside a configured workspace.
  # The subcommand list is maintained in Go (llmux is-subcommand).
  set -l _llmux_first_pos ""
  for arg in $argv
    string match -q -- '-*' $arg; and continue
    set _llmux_first_pos $arg
    break
  end
  if test -n "$_llmux_first_pos"; and %s is-subcommand $_llmux_first_pos 2>/dev/null
    set -l _llmux_pt_output (string split \n (%s resolve (pwd -P) -- $argv 2>/dev/null))
    if test $status -eq 0
      CLAUDE_CONFIG_DIR=$_llmux_pt_output[1] command claude $argv
    else
      command claude $argv
    end
    return
  end

  set -l resolve_output (string split \n (%s resolve (pwd -P) -- $argv))
  set -l resolve_status $status
  if test $resolve_status -eq 2
    %s register (pwd -P); or return 1
    set resolve_output (string split \n (%s resolve (pwd -P) -- $argv)); or return 1
  else if test $resolve_status -ne 0
    echo "llmux: no workspace configured for "(pwd -P) >&2
    echo "Run 'llmux' to manage workspaces." >&2
    return 1
  end
  set -l config_dir $resolve_output[1]
  set -l extra_flags ""
  if test (count $resolve_output) -ge 2
    set extra_flags $resolve_output[2]
  end
  set -l args
  for arg in $argv
    if test "$arg" = "--no-worktree" -o "$arg" = "-nw"
      continue
    end
    set -a args $arg
  end
  if test "$extra_flags" = "--worktree"
    set -l default_branch (git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')
    if test -n "$default_branch"
      git fetch origin $default_branch 2>/dev/null
    end
    set args --worktree $args
  end
  CLAUDE_CONFIG_DIR=$config_dir command claude $args
end`, bin, bin, bin, bin, bin)
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

	var line string
	switch sh {
	case "fish":
		line = fishEvalLine(bin, sh)
	default:
		line = evalLine(bin, sh)
	}

	if strings.Contains(content, marker) {
		// Replace existing eval block (marker + next line) with the new one
		lines := strings.Split(content, "\n")
		var result []string
		for i := 0; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == marker {
				// Skip the marker and the eval line that follows it
				i++ // skip eval line
				continue
			}
			result = append(result, lines[i])
		}
		// Remove trailing empty lines before appending
		for len(result) > 0 && result[len(result)-1] == "" {
			result = result[:len(result)-1]
		}
		content = strings.Join(result, "\n") + "\n\n" + line + "\n"
		return rc, os.WriteFile(rc, []byte(content), 0644)
	}

	// Ensure blank line before the marker for readability
	if len(content) > 0 && !strings.HasSuffix(content, "\n\n") {
		if strings.HasSuffix(content, "\n") {
			line = "\n" + line
		} else {
			line = "\n\n" + line
		}
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
