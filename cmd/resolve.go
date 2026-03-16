package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/allskar/llmux/internal/commands"
	"github.com/allskar/llmux/internal/config"
	"github.com/spf13/cobra"
)

// isGitRepo checks whether dir (or any ancestor) contains a .git entry.
func isGitRepo(dir string) bool {
	dir = filepath.Clean(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

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

		commands.Ensure()

		fmt.Fprint(os.Stderr, "\033[90m↳ llmux "+DisplayVersion()+"\033[0m\n")
		fmt.Fprint(os.Stderr, "\033[90m↳ account: "+result.WorkspaceName+"\033[0m\n")
		fmt.Print(result.SessionDir)
		if result.APIKey != "" {
			fmt.Print("\n" + result.APIKey)
		} else {
			fmt.Print("\n")
		}
		if result.Worktree {
			if isGitRepo(args[0]) {
				fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode enabled. Use --no-worktree to open claude normally.\033[0m\n")
				fmt.Print("\n--worktree")
			} else {
				fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode skipped: not a git repository.\033[0m\n")
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}
