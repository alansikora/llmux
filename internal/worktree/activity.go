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

// buildClaudeTranscriptIndex walks every profile's `projects/` directory
// once and returns a map from Claude-encoded project path to the newest
// `*.jsonl` mtime found across all profiles. Callers do a single map
// lookup per session instead of re-scanning the sessions dir per session.
//
// Claude appends to the JSONL on every turn, so the indexed mtime
// directly tracks when the user/agent last did anything in the session.
func buildClaudeTranscriptIndex() map[string]time.Time {
	index := map[string]time.Time{}
	sessionsDir := config.SessionsDir()
	profiles, err := os.ReadDir(sessionsDir)
	if err != nil {
		return index
	}

	for _, pf := range profiles {
		if !pf.IsDir() {
			continue
		}
		projectsDir := filepath.Join(sessionsDir, pf.Name(), "projects")
		projDirs, err := os.ReadDir(projectsDir)
		if err != nil {
			continue
		}
		for _, pd := range projDirs {
			if !pd.IsDir() {
				continue
			}
			entries, err := os.ReadDir(filepath.Join(projectsDir, pd.Name()))
			if err != nil {
				continue
			}
			var latest time.Time
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
			if latest.After(index[pd.Name()]) {
				index[pd.Name()] = latest
			}
		}
	}
	return index
}
