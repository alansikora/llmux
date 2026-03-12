package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/worktree"
	"github.com/spf13/cobra"
)

// mainWorktreeRoot returns the root of the main git worktree for the given
// directory. If dir is inside a linked worktree, it follows the common git dir
// back to the main repository root.
func mainWorktreeRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		// Relative .git means we're already in the main worktree.
		cmd2 := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd2.Dir = dir
		out2, err := cmd2.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out2)), nil
	}
	// Absolute path: strip the trailing /.git component.
	return filepath.Dir(gitDir), nil
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions [workspace]",
	Short: "List worktree sessions for a workspace",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ws, err := resolveWorkspace(cfg, args)
		if err != nil {
			return err
		}

		// When no explicit workspace was given, prefer the git repo root over
		// the (potentially broader) workspace path so that sessions stored in
		// {repo}/.claude/worktrees/ are found correctly.
		sessionsPath := ws.Path
		if len(args) == 0 {
			cwd, err := os.Getwd()
			if err == nil {
				if root, err := mainWorktreeRoot(cwd); err == nil {
					sessionsPath = root
				}
			}
		}

		sessions, err := worktree.ListSessions(sessionsPath)
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No worktree sessions found.")
			return nil
		}

		applied, _ := worktree.HasAppliedSession(sessionsPath)

		for _, s := range sessions {
			indicator := " "
			if s.Name == applied {
				indicator = "▶"
			}
			fmt.Printf("%s %-30s %-30s %d files changed\n", indicator, s.Name, s.Branch, s.ChangedFiles)
		}
		return nil
	},
}

func resolveWorkspace(cfg *config.Config, args []string) (*config.Workspace, error) {
	if len(args) > 0 {
		return cfg.FindWorkspace(args[0])
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return cfg.FindWorkspace(cwd)
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
}
