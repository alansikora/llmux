package skills

import (
	"errors"
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

	if err := ensureSessionSymlinks(skillsDir); err != nil {
		return "", err
	}

	return skillsDir, nil
}

// Ensure creates ~/.claude/skills if missing and keeps every session directory
// symlinked to it. Idempotent and cheap to call on every resolve; errors are
// swallowed so a transient filesystem issue can't block launching claude.
func Ensure() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	skillsDir := filepath.Join(home, ".claude", "skills")

	// MkdirAll is a no-op when the directory already exists, so we call it
	// unconditionally — a bare os.Stat guard would miss the case where
	// skillsDir is itself a symlink to a missing target.
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return
	}

	ensureSessionSymlinks(skillsDir) //nolint:errcheck // best-effort on hot path
}

// ensureSessionSymlinks creates a `skills` symlink in each session directory
// that doesn't already have a valid one pointing at skillsDir. Wrong-target
// or dangling symlinks are removed and recreated so moving ~/.claude/skills
// doesn't leave profiles permanently broken. A non-symlink entry (real file
// or directory) is left alone and reported as an error — we never destroy
// user data. Idempotent.
func ensureSessionSymlinks(skillsDir string) error {
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
		dst := filepath.Join(sessionsDir, e.Name(), "skills")
		if fi, err := os.Lstat(dst); err == nil {
			if fi.Mode()&os.ModeSymlink == 0 {
				errs = append(errs, fmt.Errorf("unexpected non-symlink at %s; refusing to overwrite", dst))
				continue
			}
			if target, err := os.Readlink(dst); err == nil {
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(dst), target)
				}
				if filepath.Clean(target) == skillsDir {
					if _, err := os.Stat(dst); err == nil {
						continue
					}
				}
			}
			if err := os.Remove(dst); err != nil {
				errs = append(errs, fmt.Errorf("removing stale symlink %s: %w", dst, err))
				continue
			}
		}
		if err := os.Symlink(skillsDir, dst); err != nil {
			errs = append(errs, fmt.Errorf("creating symlink %s: %w", dst, err))
		}
	}
	return errors.Join(errs...)
}
