package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/worktree"
	"github.com/spf13/cobra"
)

var unapplyCmd = &cobra.Command{
	Use:   "unapply",
	Short: "Revert applied worktree session changes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		// Sessions live under the git main worktree root; they're not scoped
		// by profile. `ResolveSessionsPath` walks up from cwd to find it.
		sessionsPath := worktree.ResolveSessionsPath(cwd)

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
