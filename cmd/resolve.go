package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/allskar/llmux/internal/commands"
	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/update"
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
			if errors.Is(err, config.ErrUnmapped) {
				os.Exit(2)
			}
			return err
		}

		commands.Ensure()

		versionLine := "\033[90m↳ llmux " + DisplayVersion()
		if latest := update.CheckUpdateNoticeCached(DisplayVersion()); latest != "" {
			versionLine += " \033[33m(" + latest + " available — run llmux upgrade)\033[90m"
		}
		versionLine += "\033[0m\n"
		fmt.Fprint(os.Stderr, versionLine)

		// Refresh the cache in the background (non-blocking) so the next
		// invocation has fresh data without stalling this one.
		go update.CheckLatest()
		if result.ProjectPath != "" {
			projectName := filepath.Base(result.ProjectPath)
			fmt.Fprintf(os.Stderr, "\033[90m↳ workspace: %s · project: %s\033[0m\n", result.WorkspaceName, projectName)
		} else {
			fmt.Fprintf(os.Stderr, "\033[90m↳ workspace: %s (default)\033[0m\n", result.WorkspaceName)
		}
		fmt.Print(result.SessionDir)
		if result.APIKey != "" {
			fmt.Print("\n" + result.APIKey)
		} else {
			fmt.Print("\n")
		}
		// Line 3: worktree flag (always print to keep line-based protocol stable)
		if result.Worktree && isGitRepo(args[0]) {
			fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode enabled. Use --no-worktree to open claude normally.\033[0m\n")
			fmt.Print("\n--worktree")
		} else {
			if result.Worktree {
				fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode skipped: not a git repository.\033[0m\n")
			}
			fmt.Print("\n")
		}
		// Line 4: auto mode flag (always print to keep line-based protocol stable)
		if result.AutoMode {
			fmt.Fprint(os.Stderr, "\033[90m↳ auto mode enabled\033[0m\n")
			fmt.Print("\n--enable-auto-mode")
		} else {
			fmt.Print("\n")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}
