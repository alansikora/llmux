package cmd

import (
	"fmt"

	"github.com/allskar/llmux/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces and projects",
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
			def := ""
			if ws.Name == cfg.DefaultWorkspace {
				def = " ★"
			}
			fmt.Printf("%s %s%s\n", auth, ws.Name, def)

			projects := cfg.ProjectsForWorkspace(ws.Name)
			for _, p := range projects {
				fmt.Printf("    %s\n", p.Path)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
