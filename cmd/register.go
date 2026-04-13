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
	Short:  "Register a directory to a profile",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Profiles) == 0 {
			return fmt.Errorf("no profiles configured. Run 'llmux' to create one")
		}

		dir, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		dir = filepath.Clean(dir)

		// Build options list
		options := make([]huh.Option[string], len(cfg.Profiles))
		for i, pf := range cfg.Profiles {
			label := pf.Name
			authInfo := config.GetAuthInfo(pf.Name)
			if authInfo.Authenticated {
				label = fmt.Sprintf("%s (%s)", pf.Name, authInfo.Email)
			}
			options[i] = huh.NewOption(label, pf.Name)
		}

		// Pre-select default profile
		selected := ""
		if cfg.DefaultProfile != "" {
			selected = cfg.DefaultProfile
		}

		err = huh.NewSelect[string]().
			Title(fmt.Sprintf("Select profile for %s", dir)).
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
