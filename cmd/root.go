package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "llmux",
	Short: "Claude workspace manager",
	Long:  "Manage multiple Claude Code workspaces with isolated sessions.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		m := tui.NewModel(cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func Execute() error {
	rootCmd.Version = Version
	return rootCmd.Execute()
}
