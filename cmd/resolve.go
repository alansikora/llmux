package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/allskar/llmux/internal/claude"
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

// hasWorktreeOverride returns true if the claude args contain flags that
// should prevent llmux from injecting --worktree. This centralises the
// override logic that previously lived in the shell wrapper.
func hasWorktreeOverride(claudeArgs []string) bool {
	for _, a := range claudeArgs {
		switch a {
		case "--no-worktree", "-nw", "--resume", "--continue", "-r", "-c":
			return true
		}
	}
	return false
}

var resolveCmd = &cobra.Command{
	Use:           "resolve [path] [-- claude-args...]",
	Short:         "Resolve profile for a path",
	Args:          cobra.MinimumNArgs(1),
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

		// Start the update check after early-exit checks so we don't leak a
		// goroutine on paths that call os.Exit.  On cache hit the channel is
		// pre-filled and the defer below is instant; on cache miss (once per
		// 24 h) we wait up to 2 s for the network request to complete and
		// write the cache file.
		updateCh := update.CheckUpdateNoticeAsync(DisplayVersion())
		defer func() {
			select {
			case <-updateCh:
			case <-time.After(2 * time.Second):
			}
		}()

		// Ensure session directory exists before EnsureSessionSymlinks runs,
		// so commands are available on the very first launch of a new profile.
		os.MkdirAll(result.SessionDir, 0755) //nolint:errcheck

		// Sync managed settings (statusLine, autoMode, etc.) to settings.json
		config.SyncProfileSettings(cfg, result.ProfileName) //nolint:errcheck

		commands.Ensure()

		versionLine := "\033[90m↳ llmux " + DisplayVersion()
		if latest := update.CheckUpdateNoticeCached(DisplayVersion()); latest != "" {
			versionLine += " \033[33m(" + latest + " available — run llmux upgrade)\033[90m"
		}
		versionLine += "\033[0m\n"
		fmt.Fprint(os.Stderr, versionLine)
		if result.ProjectPath != "" {
			projectName := filepath.Base(result.ProjectPath)
			fmt.Fprintf(os.Stderr, "\033[90m↳ profile: %s · project: %s\033[0m\n", result.ProfileName, projectName)
		} else {
			fmt.Fprintf(os.Stderr, "\033[90m↳ profile: %s (default)\033[0m\n", result.ProfileName)
		}
		// Line 1: session directory
		fmt.Println(result.SessionDir)

		// Line 2: extra CLI flags (worktree or empty)
		claudeArgs := args[1:] // everything after the path (passed via --)
		subcmd := claude.IsSubcommand(claudeArgs)
		worktreeOverride := hasWorktreeOverride(claudeArgs)
		if !subcmd && !worktreeOverride && result.Worktree && isGitRepo(args[0]) {
			fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode enabled. Use --no-worktree to open claude normally.\033[0m\n")
			fmt.Print("--worktree")
		} else if result.Worktree && !subcmd && !worktreeOverride {
			fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode skipped: not a git repository.\033[0m\n")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}
