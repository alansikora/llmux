package cmd

import (
	"fmt"

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

		if err := worktree.Apply(ws.Path, args[0]); err != nil {
			return err
		}

		fmt.Printf("Applied session %q to %s\n", args[0], ws.Path)
		return nil
	},
}

var applyWorkspace string

func init() {
	applyCmd.Flags().StringVarP(&applyWorkspace, "workspace", "w", "", "workspace name")
	rootCmd.AddCommand(applyCmd)
}
