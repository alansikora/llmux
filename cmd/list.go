package cmd

import (
	"fmt"

	"github.com/allskar/llmux/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Workspaces) == 0 {
			fmt.Println("No workspaces configured. Run 'llmux' to add one.")
			return nil
		}

		for _, ws := range cfg.Workspaces {
			auth := "○"
			if config.IsAuthenticated(ws.Name) {
				auth = "●"
			}
			fmt.Printf("%s %s\t%s\n", auth, ws.Name, ws.Path)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
