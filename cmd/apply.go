package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/worktree"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply <session>",
	Short: "Apply worktree session changes to the main working tree",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		var wsArgs []string
		if applyWorkspace != "" {
			wsArgs = []string{applyWorkspace}
		}
		ws, err := resolveWorkspace(cfg, wsArgs)
		if err != nil {
			return err
		}

		// When no explicit workspace was given, prefer the git repo root so
		// that sessions stored in {repo}/.claude/worktrees/ are found correctly.
		sessionsPath := ws.Path
		if applyWorkspace == "" {
			cwd, err := os.Getwd()
			if err == nil {
				sessionsPath = worktree.ResolveSessionsPath(cwd)
			}
		}

		if err := worktree.Apply(sessionsPath, args[0], cfg.ApplyMarker); err != nil {
			return err
		}

		fmt.Printf("Applied session %q to %s\n", args[0], sessionsPath)
		return nil
	},
}

var applyWorkspace string

func init() {
	applyCmd.Flags().StringVarP(&applyWorkspace, "workspace", "w", "", "workspace name")
	rootCmd.AddCommand(applyCmd)
}
