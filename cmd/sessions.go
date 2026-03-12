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

		ws, err := resolveWorkspace(cfg, args)
		if err != nil {
			return err
		}

		sessions, err := worktree.ListSessions(ws.Path)
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No worktree sessions found.")
			return nil
		}

		applied, _ := worktree.HasAppliedSession(ws.Path)

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
