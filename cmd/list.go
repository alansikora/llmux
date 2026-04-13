package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/allskar/llmux/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects with their profile and auth status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Projects) == 0 {
			fmt.Println("No projects configured. Run 'llmux' to add one.")
			return nil
		}

		// Sort projects alphabetically by base name (case-insensitive)
		sorted := make([]config.Project, len(cfg.Projects))
		copy(sorted, cfg.Projects)
		sort.Slice(sorted, func(i, j int) bool {
			return strings.ToLower(filepath.Base(sorted[i].Path)) < strings.ToLower(filepath.Base(sorted[j].Path))
		})

		// Determine column width for alignment
		maxName := 0
		for _, p := range sorted {
			if n := len(filepath.Base(p.Path)); n > maxName {
				maxName = n
			}
		}

		for _, p := range sorted {
			auth := "○"
			if config.IsAuthenticated(p.Profile) {
				auth = "●"
			}
			name := filepath.Base(p.Path)
			fmt.Printf("%s %-*s  %s  (%s)\n", auth, maxName, name, p.Path, p.Profile)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
