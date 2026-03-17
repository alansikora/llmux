package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/allskar/llmux/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:    "register [path]",
	Short:  "Register a directory to a workspace",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Workspaces) == 0 {
			return fmt.Errorf("no workspaces configured. Run 'llmux' to create one")
		}

		dir, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		dir = filepath.Clean(dir)

		// Build options list
		options := make([]huh.Option[string], len(cfg.Workspaces))
		for i, ws := range cfg.Workspaces {
			label := ws.Name
			authInfo := config.GetAuthInfo(ws.Name)
			if authInfo.Authenticated {
				label = fmt.Sprintf("%s (%s)", ws.Name, authInfo.Email)
			}
			options[i] = huh.NewOption(label, ws.Name)
		}

		// Pre-select default workspace
		selected := ""
		if cfg.DefaultWorkspace != "" {
			selected = cfg.DefaultWorkspace
		}

		err = huh.NewSelect[string]().
			Title(fmt.Sprintf("Select workspace for %s", dir)).
			Options(options...).
			Value(&selected).
			Run()
		if err != nil {
			return err
		}

		if err := cfg.AddProject(dir, selected); err != nil {
			return err
		}

		return config.Save(cfg)
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
}
