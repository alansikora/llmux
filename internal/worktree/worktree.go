package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const StaleDuration = 7 * 24 * time.Hour

type Session struct {
	Name          string
	Branch        string
	ChangedFiles  int
	LastCommit    time.Time
	LastClaudeRun time.Time
	Path          string
	WorkspacePath string
}

// LastActivity is the most recent signal of activity in the session:
// either a git commit on its branch or a Claude turn writing to the
// session transcript. This is what should drive staleness and sort order —
// a session can be actively in use with no recent commits.
func (s Session) LastActivity() time.Time {
	if s.LastClaudeRun.After(s.LastCommit) {
		return s.LastClaudeRun
	}
	return s.LastCommit
}

func (s Session) IsStale() bool {
	t := s.LastActivity()
	return !t.IsZero() && time.Since(t) > StaleDuration
}

func WorktreesDir(workspacePath string) string {
	return filepath.Join(workspacePath, ".claude", "worktrees")
}

// DetectCurrentSession checks if cwd is inside a .claude/worktrees/{name}/
// directory and returns the session name if so.
func DetectCurrentSession(cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	parts := strings.Split(filepath.ToSlash(abs), "/")
	for i := len(parts) - 1; i >= 2; i-- {
		if parts[i-1] == "worktrees" && parts[i-2] == ".claude" {
			return parts[i], nil
		}
	}
	return "", fmt.Errorf("not inside a worktree session")
}

func ListSessions(workspacePath string) ([]Session, error) {
	dir := WorktreesDir(workspacePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return listSessionsInSubdirs(workspacePath)
		}
		return nil, err
	}

	// Collect candidate directories first.
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		wtPath := filepath.Join(dir, entry.Name())
		// Verify it's a git worktree by checking for .git file
		if _, err := os.Stat(filepath.Join(wtPath, ".git")); err != nil {
			continue
		}
		names = append(names, entry.Name())
	}

	// Fetch per-session metadata in parallel. Each session requires 3
	// sequential git calls (branch, diff-stat, last-activity), so we run
	// one goroutine per session to maximise concurrency.
	results := make([]*Session, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			wtPath := filepath.Join(dir, name)

			branch, err := runGit(wtPath, "rev-parse", "--abbrev-ref", "HEAD")
			if err != nil {
				return
			}
			trimmedBranch := strings.TrimSpace(branch)

			changedFiles := 0
			if stat, err := runGit(workspacePath, "diff", "--stat", "HEAD..."+trimmedBranch); err == nil {
				lines := strings.Split(strings.TrimSpace(stat), "\n")
				if len(lines) > 0 {
					// Last line is summary like " 3 files changed, 10 insertions(+), 2 deletions(-)"
					summary := lines[len(lines)-1]
					if parts := strings.Fields(summary); len(parts) >= 1 {
						if n, err := strconv.Atoi(parts[0]); err == nil {
							changedFiles = n
						}
					}
				}
			}

			var lastCommit time.Time
			if ts, err := runGit(wtPath, "log", "-1", "--format=%ct"); err == nil {
				if epoch, err := strconv.ParseInt(strings.TrimSpace(ts), 10, 64); err == nil {
					lastCommit = time.Unix(epoch, 0)
				}
			}

			results[i] = &Session{
				Name:          name,
				Branch:        trimmedBranch,
				ChangedFiles:  changedFiles,
				LastCommit:    lastCommit,
				LastClaudeRun: latestClaudeTranscriptMtime(wtPath),
				Path:          wtPath,
				WorkspacePath: workspacePath,
			}
		}(i, name)
	}
	wg.Wait()

	// Preserve directory order from the parallel fetch, dropping any that
	// failed the branch lookup (nil entries).
	sessions := make([]Session, 0, len(results))
	for _, s := range results {
		if s != nil {
			sessions = append(sessions, *s)
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity().After(sessions[j].LastActivity())
	})

	return sessions, nil
}

func HasAppliedSession(workspacePath string) (string, bool) {
	state, err := LoadState(workspacePath)
	if err != nil || state == nil {
		return "", false
	}
	return state.Session, true
}

const MarkerFileName = ".llmux-applied"

func writeMarker(workspacePath, sessionName, branch string) error {
	content := fmt.Sprintf("session: %s\nbranch: %s\n", sessionName, branch)
	return os.WriteFile(filepath.Join(workspacePath, MarkerFileName), []byte(content), 0644)
}

func removeMarker(workspacePath string) {
	os.Remove(filepath.Join(workspacePath, MarkerFileName))
}

func Apply(workspacePath, sessionName string, applyMarker ...bool) error {
	if applied, ok := HasAppliedSession(workspacePath); ok {
		return fmt.Errorf("session %q is already applied; run 'llmux unapply' first", applied)
	}

	session, err := findSession(workspacePath, sessionName)
	if err != nil {
		return err
	}

	// Check for dirty working tree and auto-stash
	status, err := runGit(workspacePath, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("checking working tree: %w", err)
	}
	stashCreated := false
	if strings.TrimSpace(status) != "" {
		if _, err := runGit(workspacePath, "stash", "push", "-m", "llmux-auto-stash"); err != nil {
			return fmt.Errorf("stashing changes: %w", err)
		}
		stashCreated = true
	}

	// Pop stash on any failure after this point
	applied := false
	defer func() {
		if !applied && stashCreated {
			if _, err := runGit(workspacePath, "stash", "pop"); err != nil {
				fmt.Fprintf(os.Stderr, "llmux: warning: failed to restore stash — run 'git stash pop' manually: %v\n", err)
			}
		}
	}()

	// Snapshot the session's full working state (committed + staged + unstaged)
	// git add -N marks untracked files so they appear in diffs.
	// Always restore the index afterwards via git reset, even on error.
	if _, err := runGit(session.Path, "add", "-N", "."); err != nil {
		return fmt.Errorf("staging untracked files for snapshot: %w", err)
	}
	// git stash create returns a ref capturing everything, without side effects
	stashOut, stashErr := runGit(session.Path, "stash", "create")
	// Always restore the session's original index state to avoid leaving
	// stale intent-to-add entries from git add -N.
	if _, resetErr := runGit(session.Path, "reset"); resetErr != nil {
		fmt.Fprintf(os.Stderr, "llmux: warning: failed to reset session index: %v\n", resetErr)
	}

	ref := strings.TrimSpace(stashOut)
	if stashErr != nil || ref == "" {
		ref = session.Branch // no uncommitted changes, use branch as-is
	}

	// Generate diff from main HEAD to the full snapshot
	diff, err := runGit(workspacePath, "diff", "HEAD", ref)
	if err != nil {
		return fmt.Errorf("generating diff: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		return fmt.Errorf("no changes to apply from session %q", sessionName)
	}

	// Apply the diff
	cmd := exec.Command("git", "apply", "--3way")
	cmd.Dir = workspacePath
	cmd.Stdin = strings.NewReader(diff)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("applying diff: %s\n%s", err, string(output))
	}

	// Save diff for reverse-apply during unapply
	if err := SaveDiff(workspacePath, diff); err != nil {
		// Reverse the already-applied diff to avoid inconsistent state
		reverseCmd := exec.Command("git", "apply", "--reverse")
		reverseCmd.Dir = workspacePath
		reverseCmd.Stdin = strings.NewReader(diff)
		reverseCmd.Run() //nolint:errcheck
		return fmt.Errorf("saving diff: %w", err)
	}

	// Save state
	if err := SaveState(workspacePath, ApplyState{
		Session:      sessionName,
		StashCreated: stashCreated,
		AppliedAt:    time.Now(),
	}); err != nil {
		// Reverse the applied diff and clean up the orphaned diff file
		reverseCmd := exec.Command("git", "apply", "--reverse")
		reverseCmd.Dir = workspacePath
		reverseCmd.Stdin = strings.NewReader(diff)
		reverseCmd.Run() //nolint:errcheck
		RemoveDiff(workspacePath)
		return fmt.Errorf("saving state: %w", err)
	}

	// Write marker file if enabled
	if len(applyMarker) > 0 && applyMarker[0] {
		writeMarker(workspacePath, sessionName, session.Branch)
	}

	applied = true
	return nil
}

func Unapply(workspacePath string) error {
	state, err := LoadState(workspacePath)
	if err != nil {
		return fmt.Errorf("reading state: %w", err)
	}
	if state == nil {
		return fmt.Errorf("no session is currently applied")
	}

	// Reverse-apply the saved diff to undo only the applied changes,
	// preserving any edits the user made after applying.
	diff, err := LoadDiff(workspacePath)
	if err != nil {
		// Fall back to destructive unapply if diff file is missing
		// (e.g. applied with an older version that didn't save diffs)
		return unapplyDestructive(workspacePath, state)
	}

	// Unstage any staged changes first
	if _, err := runGit(workspacePath, "reset", "HEAD"); err != nil {
		return fmt.Errorf("unstaging changes: %w", err)
	}

	cmd := exec.Command("git", "apply", "--reverse")
	cmd.Dir = workspacePath
	cmd.Stdin = strings.NewReader(diff)
	output, applyErr := cmd.CombinedOutput()
	if applyErr != nil {
		return fmt.Errorf("reverse-apply failed (manual resolution needed): %s\n%s", applyErr, string(output))
	}

	// Remove marker and diff only after reverse-apply succeeds
	removeMarker(workspacePath)
	RemoveDiff(workspacePath)

	// Pop stash if one was created
	if state.StashCreated {
		if _, err := runGit(workspacePath, "stash", "pop"); err != nil {
			return fmt.Errorf("restoring stash: %w", err)
		}
	}

	return RemoveState(workspacePath)
}

// unapplyDestructive is the legacy fallback when no saved diff is available.
func unapplyDestructive(workspacePath string, state *ApplyState) error {
	if _, err := runGit(workspacePath, "reset", "HEAD"); err != nil {
		return fmt.Errorf("unstaging changes: %w", err)
	}
	if _, err := runGit(workspacePath, "checkout", "."); err != nil {
		return fmt.Errorf("reverting changes: %w", err)
	}
	if _, err := runGit(workspacePath, "clean", "-fd"); err != nil {
		return fmt.Errorf("cleaning untracked files: %w", err)
	}
	if state.StashCreated {
		if _, err := runGit(workspacePath, "stash", "pop"); err != nil {
			return fmt.Errorf("restoring stash: %w", err)
		}
	}
	return RemoveState(workspacePath)
}

// ResolveSessionsPath returns the git main worktree root for the given
// directory, falling back to dir itself if git is unavailable.
func ResolveSessionsPath(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return dir
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		// Relative .git means we're already in the main worktree.
		cmd2 := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd2.Dir = dir
		out2, err := cmd2.Output()
		if err != nil {
			return dir
		}
		return strings.TrimSpace(string(out2))
	}
	// Absolute path: strip the trailing /.git component.
	return filepath.Dir(gitDir)
}

func listSessionsInSubdirs(parentPath string) ([]Session, error) {
	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return nil, nil
	}
	// Track the roots we've already descended into so that symlinks or
	// unusual layouts can't cause `ListSessions` to recurse back into the
	// parent (or any sibling whose resolved root matches another).
	seen := map[string]bool{parentPath: true}
	var all []Session
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		subPath := filepath.Join(parentPath, entry.Name())
		if _, err := os.Stat(filepath.Join(subPath, ".git")); err != nil {
			continue
		}
		repoRoot := ResolveSessionsPath(subPath)
		if seen[repoRoot] {
			continue
		}
		seen[repoRoot] = true
		sessions, err := ListSessions(repoRoot)
		if err != nil {
			continue
		}
		all = append(all, sessions...)
	}
	return all, nil
}

// findSession lists sessions in workspacePath and returns the one matching name.
func findSession(workspacePath, name string) (*Session, error) {
	sessions, err := ListSessions(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	for i := range sessions {
		if sessions[i].Name == name {
			return &sessions[i], nil
		}
	}
	return nil, fmt.Errorf("session %q not found", name)
}

func FindAppliedWorkspace(sessions []Session) (workspacePath, sessionName string, ok bool) {
	seen := map[string]bool{}
	for _, s := range sessions {
		if s.WorkspacePath == "" || seen[s.WorkspacePath] {
			continue
		}
		seen[s.WorkspacePath] = true
		if name, applied := HasAppliedSession(s.WorkspacePath); applied {
			return s.WorkspacePath, name, true
		}
	}
	return "", "", false
}

func Delete(workspacePath, sessionName string, force bool) error {
	session, err := findSession(workspacePath, sessionName)
	if err != nil {
		return err
	}
	args := []string{"worktree", "remove", session.Path}
	if force {
		args = append(args, "--force")
	}
	if _, err := runGit(session.WorkspacePath, args...); err != nil {
		return err
	}
	return nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output), nil
}
