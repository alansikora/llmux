package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/worktree"
	"github.com/spf13/cobra"
)

var unapplyCmd = &cobra.Command{
	Use:   "unapply",
	Short: "Revert applied worktree session changes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ws, err := resolveWorkspace(cfg, nil)
		if err != nil {
			return err
		}

		sessionsPath := ws.Path
		cwd, err := os.Getwd()
		if err == nil {
			sessionsPath = worktree.ResolveSessionsPath(cwd)
		}

		if err := worktree.Unapply(sessionsPath); err != nil {
			return err
		}

		fmt.Printf("Unapplied session changes from %s\n", sessionsPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unapplyCmd)
}
