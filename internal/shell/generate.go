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
  local resolve_output config_dir api_key
  resolve_output="$(%s resolve "$(pwd -P)")"
  if [ $? -ne 0 ]; then
    echo "llmux: no workspace configured for $(pwd -P)" >&2
    echo "Run 'llmux' to manage workspaces." >&2
    return 1
  fi
  config_dir="$(echo "$resolve_output" | head -n1)"
  api_key="$(echo "$resolve_output" | sed -n '2p')"
  if [ -n "$api_key" ]; then
    ANTHROPIC_API_KEY="$api_key" CLAUDE_CONFIG_DIR="$config_dir" command claude "$@"
  else
    CLAUDE_CONFIG_DIR="$config_dir" command claude "$@"
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
  if test (count $resolve_output) -ge 2; and test -n "$resolve_output[2]"
    ANTHROPIC_API_KEY=$resolve_output[2] CLAUDE_CONFIG_DIR=$config_dir command claude $argv
  else
    CLAUDE_CONFIG_DIR=$config_dir command claude $argv
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
func Generate(bin, sh string) (string, error) {
	switch sh {
	case "zsh", "bash":
		return snippet(bin), nil
	case "fish":
		return fishSnippet(bin), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: zsh, bash, fish)", sh)
	}
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
