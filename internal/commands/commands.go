package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/allskar/llmux/internal/config"
)

var commands = map[string]string{
	"apply": `Run ` + "`llmux apply`" + ` to apply this worktree session's changes to the main workspace. The command auto-detects the current session name from the working directory.

If it fails because another session is already applied, tell the user to run ` + "`/unapply`" + ` first.
`,
	"unapply": `Run ` + "`llmux unapply`" + ` to revert previously applied worktree session changes from the main workspace.

If no session is currently applied, let the user know.
`,
}

// Install writes the llmux slash commands to ~/.claude/commands/llmux/
// and ensures all existing session directories have a symlink to
// ~/.claude/commands so commands are available in every workspace.
// Returns the directory path where commands were installed.
func Install() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	commandsDir := filepath.Join(home, ".claude", "commands")
	dir := filepath.Join(commandsDir, "llmux")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating commands directory: %w", err)
	}

	// Clean up old top-level commands from previous installs
	for name := range commands {
		os.Remove(filepath.Join(commandsDir, name+".md"))
	}

	for name, content := range commands {
		path := filepath.Join(dir, name+".md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("writing %s: %w", name, err)
		}
	}

	// Ensure all existing session directories have the commands symlink.
	// This covers workspaces that were resolved before commands were installed
	// (e.g. after an upgrade).
	EnsureSessionSymlinks(commandsDir)

	return dir, nil
}

// Ensure installs commands if missing and ensures all session symlinks exist.
// It is idempotent and cheap to call on every resolve.
func Ensure() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	commandsDir := filepath.Join(home, ".claude", "commands")
	dir := filepath.Join(commandsDir, "llmux")

	// Install command files if the directory doesn't exist yet
	// (e.g. binary was upgraded but `llmux init` wasn't re-run).
	if _, err := os.Stat(dir); err != nil {
		Install() //nolint: errcheck
	}

	EnsureSessionSymlinks(commandsDir)
}

// EnsureSessionSymlinks creates commands symlinks in all existing session
// directories that don't already have one. It is idempotent and safe to call
// on every resolve (i.e. every `claude` invocation).
func EnsureSessionSymlinks(commandsDir string) {
	sessionsDir := filepath.Join(config.ConfigDir(), "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dst := filepath.Join(sessionsDir, e.Name(), "commands")
		if _, err := os.Lstat(dst); err == nil {
			continue // already exists
		}
		os.Symlink(commandsDir, dst)
	}
}
