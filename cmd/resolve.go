package cmd

import (
	"fmt"

	"github.com/allskar/llmux/internal/config"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:           "resolve [path]",
	Short:         "Resolve workspace for a path",
	Args:          cobra.ExactArgs(1),
	Hidden:        true,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		result, err := cfg.Resolve(args[0])
		if err != nil {
			return err
		}

		fmt.Print(result.SessionDir)
		if result.APIKey != "" {
			fmt.Print("\n" + result.APIKey)
		} else {
			fmt.Print("\n")
		}
		if result.Worktree {
			fmt.Print("\n--worktree")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}
