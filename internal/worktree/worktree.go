package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const StaleDuration = 7 * 24 * time.Hour

type Session struct {
	Name          string
	Branch        string
	ChangedFiles  int
	LastActivity  time.Time
	Path          string
	WorkspacePath string
}

func (s Session) IsStale() bool {
	return !s.LastActivity.IsZero() && time.Since(s.LastActivity) > StaleDuration
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

	var sessions []Session
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		wtPath := filepath.Join(dir, entry.Name())

		// Verify it's a git worktree by checking for .git file
		gitPath := filepath.Join(wtPath, ".git")
		if _, err := os.Stat(gitPath); err != nil {
			continue
		}

		branch, err := runGit(wtPath, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			continue
		}

		trimmedBranch := strings.TrimSpace(branch)

		changedFiles := 0
		stat, err := runGit(workspacePath, "diff", "--stat", "HEAD..."+trimmedBranch)
		if err == nil {
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

		var lastActivity time.Time
		if ts, err := runGit(wtPath, "log", "-1", "--format=%ct"); err == nil {
			if epoch, err := strconv.ParseInt(strings.TrimSpace(ts), 10, 64); err == nil {
				lastActivity = time.Unix(epoch, 0)
			}
		}

		sessions = append(sessions, Session{
			Name:          entry.Name(),
			Branch:        trimmedBranch,
			ChangedFiles:  changedFiles,
			LastActivity:  lastActivity,
			Path:          wtPath,
			WorkspacePath: workspacePath,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
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

	// Find the session and its branch
	sessions, err := ListSessions(workspacePath)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	var session *Session
	for i := range sessions {
		if sessions[i].Name == sessionName {
			session = &sessions[i]
			break
		}
	}
	if session == nil {
		return fmt.Errorf("session %q not found", sessionName)
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

	// Generate and apply diff
	diff, err := runGit(workspacePath, "diff", "HEAD..."+session.Branch)
	if err != nil {
		if stashCreated {
			runGit(workspacePath, "stash", "pop") //nolint:errcheck
		}
		return fmt.Errorf("generating diff: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		if stashCreated {
			runGit(workspacePath, "stash", "pop") //nolint:errcheck
		}
		return fmt.Errorf("no changes to apply from session %q", sessionName)
	}

	// Apply the diff
	cmd := exec.Command("git", "apply", "--3way")
	cmd.Dir = workspacePath
	cmd.Stdin = strings.NewReader(diff)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if stashCreated {
			runGit(workspacePath, "stash", "pop") //nolint:errcheck
		}
		return fmt.Errorf("applying diff: %s\n%s", err, string(output))
	}

	// Save state
	if err := SaveState(workspacePath, ApplyState{
		Session:      sessionName,
		StashCreated: stashCreated,
		AppliedAt:    time.Now(),
	}); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	// Write marker file if enabled
	if len(applyMarker) > 0 && applyMarker[0] {
		writeMarker(workspacePath, sessionName, session.Branch)
	}

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

	// Remove marker file if present
	removeMarker(workspacePath)

	// Unstage any staged changes (git apply --3way stages files)
	if _, err := runGit(workspacePath, "reset", "HEAD"); err != nil {
		return fmt.Errorf("unstaging changes: %w", err)
	}

	// Discard applied changes
	if _, err := runGit(workspacePath, "checkout", "."); err != nil {
		return fmt.Errorf("reverting changes: %w", err)
	}

	// Clean any untracked files that were added by the apply
	if _, err := runGit(workspacePath, "clean", "-fd"); err != nil {
		return fmt.Errorf("cleaning untracked files: %w", err)
	}

	// Pop stash if one was created
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
		sessions, err := ListSessions(repoRoot)
		if err != nil {
			continue
		}
		all = append(all, sessions...)
	}
	return all, nil
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
	sessions, err := ListSessions(workspacePath)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}
	var session *Session
	for i := range sessions {
		if sessions[i].Name == sessionName {
			session = &sessions[i]
			break
		}
	}
	if session == nil {
		return fmt.Errorf("session %q not found", sessionName)
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
