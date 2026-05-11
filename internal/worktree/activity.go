package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allskar/llmux/internal/config"
)

// encodeClaudeProjectPath converts an absolute filesystem path into the
// directory name Claude Code uses under `<session-dir>/projects/`. The
// encoding replaces both `/` and `.` with `-`, matching Claude Code's
// convention (e.g. `/home/u/.claude/worktrees/foo` →
// `-home-u--claude-worktrees-foo`).
func encodeClaudeProjectPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.ReplaceAll(p, "/", "-")
	p = strings.ReplaceAll(p, ".", "-")
	return p
}

// latestClaudeTranscriptMtime returns the most recent mtime among Claude
// transcript files (`*.jsonl`) for the given worktree path, across every
// llmux profile. Returns the zero time if no transcript directory exists.
//
// Claude appends to the JSONL on every turn, so this mtime directly tracks
// when the user/agent last did anything in the session.
func latestClaudeTranscriptMtime(worktreePath string) time.Time {
	encoded := encodeClaudeProjectPath(worktreePath)
	profiles, err := os.ReadDir(config.SessionsDir())
	if err != nil {
		return time.Time{}
	}

	var latest time.Time
	for _, pf := range profiles {
		if !pf.IsDir() {
			continue
		}
		projDir := filepath.Join(config.SessionsDir(), pf.Name(), "projects", encoded)
		entries, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
	}
	return latest
}
