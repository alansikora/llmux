package cmd

import (
	"os"

	"github.com/allskar/llmux/internal/claude"
	"github.com/spf13/cobra"
)

var isSubcommandCmd = &cobra.Command{
	Use:           "is-subcommand [name]",
	Short:         "Check if a name is a claude subcommand",
	Args:          cobra.ExactArgs(1),
	Hidden:        true,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		if !claude.IsSubcommand(args) {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(isSubcommandCmd)
}
