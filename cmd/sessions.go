package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/worktree"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions [workspace]",
	Short: "List worktree sessions for a workspace",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ws, _, err := resolveWorkspace(cfg, args)
		if err != nil {
			return err
		}

		// Prefer git repo root for finding sessions
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		sessionsPath := worktree.ResolveSessionsPath(cwd)

		_ = ws // workspace used for session dir context if needed

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

// resolveWorkspace finds the workspace (and optionally project) for the given args.
// If args has a name, looks up by workspace name. Otherwise uses cwd to find project → workspace.
func resolveWorkspace(cfg *config.Config, args []string) (*config.Workspace, *config.Project, error) {
	if len(args) > 0 {
		ws, err := cfg.FindWorkspace(args[0])
		return ws, nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	return cfg.FindWorkspaceForDir(cwd)
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
}
