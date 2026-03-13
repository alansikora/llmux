package commands

import (
	"fmt"
	"os"
	"path/filepath"
)

var commands = map[string]string{
	"apply": `Run ` + "`llmux apply`" + ` to apply this worktree session's changes to the main workspace. The command auto-detects the current session name from the working directory.

If it fails because another session is already applied, tell the user to run ` + "`/unapply`" + ` first.
`,
	"unapply": `Run ` + "`llmux unapply`" + ` to revert previously applied worktree session changes from the main workspace.

If no session is currently applied, let the user know.
`,
}

// Install writes the llmux slash commands to ~/.claude/commands/llmux/.
// Returns the directory path where commands were installed.
func Install() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".claude", "commands", "llmux")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating commands directory: %w", err)
	}

	// Clean up old top-level commands from previous installs
	oldDir := filepath.Join(home, ".claude", "commands")
	for name := range commands {
		os.Remove(filepath.Join(oldDir, name+".md"))
	}

	for name, content := range commands {
		path := filepath.Join(dir, name+".md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("writing %s: %w", name, err)
		}
	}

	return dir, nil
}
