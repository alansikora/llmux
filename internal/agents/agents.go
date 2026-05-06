package agents

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/allskar/llmux/internal/config"
)

// Install ensures ~/.claude/agents exists and creates the agents symlink in
// every existing session directory. Because llmux ships no agents of its own,
// this is pure plumbing: each profile sees the user's global agents directory
// through its own CLAUDE_CONFIG_DIR.
// Returns the directory path that session symlinks target.
func Install() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	agentsDir := filepath.Join(home, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return "", fmt.Errorf("creating agents directory: %w", err)
	}

	if err := ensureSessionSymlinks(agentsDir); err != nil {
		return "", err
	}

	return agentsDir, nil
}

// Ensure creates ~/.claude/agents if missing and keeps every session directory
// symlinked to it. Idempotent and cheap to call on every resolve; errors are
// swallowed so a transient filesystem issue can't block launching claude.
func Ensure() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	agentsDir := filepath.Join(home, ".claude", "agents")

	// MkdirAll is a no-op when the directory already exists, so we call it
	// unconditionally — a bare os.Stat guard would miss the case where
	// agentsDir is itself a symlink to a missing target.
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return
	}

	ensureSessionSymlinks(agentsDir) //nolint:errcheck // best-effort on hot path
}

// ensureSessionSymlinks creates an `agents` symlink in each session directory
// that doesn't already have a valid one. Dangling symlinks (target missing)
// are removed and recreated so moving ~/.claude/agents doesn't leave profiles
// permanently broken. Idempotent.
func ensureSessionSymlinks(agentsDir string) error {
	sessionsDir := config.SessionsDir()
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading sessions directory: %w", err)
	}

	var errs []error
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dst := filepath.Join(sessionsDir, e.Name(), "agents")
		if _, err := os.Lstat(dst); err == nil {
			// Something exists at dst. If os.Stat resolves it, leave it
			// alone; otherwise it's a dangling symlink — replace it.
			if _, err := os.Stat(dst); err == nil {
				continue
			}
			if err := os.Remove(dst); err != nil {
				errs = append(errs, fmt.Errorf("removing stale symlink %s: %w", dst, err))
				continue
			}
		}
		if err := os.Symlink(agentsDir, dst); err != nil {
			errs = append(errs, fmt.Errorf("creating symlink %s: %w", dst, err))
		}
	}
	return errors.Join(errs...)
}
