package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/update"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade llmux to the latest version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		current := DisplayVersion()

		fmt.Fprintf(os.Stderr, "Current version: %s\n", current)
		fmt.Fprintf(os.Stderr, "Checking for updates...\n")

		latest, err := update.CheckLatest()
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}

		if !update.IsNewer(current, latest) {
			fmt.Fprintf(os.Stderr, "Already up to date.\n")
			return nil
		}

		fmt.Fprintf(os.Stderr, "Upgrading to %s...\n", latest)
		if err := update.Upgrade(latest); err != nil {
			return fmt.Errorf("upgrade failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Successfully upgraded to %s\n", latest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
