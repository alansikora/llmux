package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/allskar/llmux/internal/config"
)

// Install ensures ~/.claude/skills exists and creates the skills symlink in
// every existing session directory. Because llmux ships no skills of its own,
// this is pure plumbing: each profile sees the user's global skills directory
// through its own CLAUDE_CONFIG_DIR.
// Returns the directory path that session symlinks target.
func Install() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	skillsDir := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return "", fmt.Errorf("creating skills directory: %w", err)
	}

	EnsureSessionSymlinks(skillsDir)

	return skillsDir, nil
}

// Ensure creates ~/.claude/skills if missing and keeps every session directory
// symlinked to it. Idempotent and cheap to call on every resolve.
func Ensure() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	skillsDir := filepath.Join(home, ".claude", "skills")

	if _, err := os.Stat(skillsDir); err != nil {
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return
		}
	}

	EnsureSessionSymlinks(skillsDir)
}

// EnsureSessionSymlinks creates a `skills` symlink in each session directory
// that doesn't already have one. Idempotent and safe to call on every resolve.
func EnsureSessionSymlinks(skillsDir string) {
	sessionsDir := config.SessionsDir()
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dst := filepath.Join(sessionsDir, e.Name(), "skills")
		if _, err := os.Lstat(dst); err == nil {
			continue // already exists
		}
		os.Symlink(skillsDir, dst)
	}
}
