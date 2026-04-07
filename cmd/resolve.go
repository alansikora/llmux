package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

const subcmdCacheFile = "claude-subcommands.json"

type subcmdCache struct {
	Subcommands []string  `json:"subcommands"`
	CachedAt    time.Time `json:"cached_at"`
}

func subcmdCachePath() string {
	return filepath.Join(config.ConfigDir(), subcmdCacheFile)
}

// readSubcmdCache returns the cached subcommand set if fresh (within TTL).
func readSubcmdCache() map[string]bool {
	return readSubcmdCacheWithTTL(24 * time.Hour)
}

// readSubcmdCacheStale returns the cached subcommand set regardless of age,
// used as a fallback when the live fetch fails.
func readSubcmdCacheStale() map[string]bool {
	return readSubcmdCacheWithTTL(0)
}

func readSubcmdCacheWithTTL(ttl time.Duration) map[string]bool {
	data, err := os.ReadFile(subcmdCachePath())
	if err != nil {
		return nil
	}
	var c subcmdCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	if ttl > 0 && time.Since(c.CachedAt) > ttl {
		return nil
	}
	if len(c.Subcommands) == 0 {
		return nil
	}
	m := make(map[string]bool, len(c.Subcommands))
	for _, s := range c.Subcommands {
		m[s] = true
	}
	return m
}

func writeSubcmdCache(subcmds map[string]bool) {
	list := make([]string, 0, len(subcmds))
	for s := range subcmds {
		list = append(list, s)
	}
	data, err := json.Marshal(subcmdCache{Subcommands: list, CachedAt: time.Now()})
	if err != nil {
		return
	}
	p := subcmdCachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0644)
}

// parseClaudeSubcommands parses the Commands section from `claude --help` output.
func parseClaudeSubcommands(output string) map[string]bool {
	subcmds := make(map[string]bool)
	lines := strings.Split(output, "\n")
	inCommands := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "Commands:" {
			inCommands = true
			continue
		}
		if !inCommands {
			continue
		}
		// Stop at the next section header: a non-indented, non-empty line
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// First field is the command name, possibly with "|" aliases
		// e.g. "plugin|plugins", "update|upgrade"
		name := strings.Fields(trimmed)[0]
		for _, n := range strings.Split(name, "|") {
			subcmds[n] = true
		}
	}
	return subcmds
}

// getClaudeSubcommands returns the set of claude CLI subcommands, using a
// cached result when available (24h TTL) to avoid spawning a subprocess on
// every invocation.
func getClaudeSubcommands() map[string]bool {
	if cached := readSubcmdCache(); cached != nil {
		return cached
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "claude", "--help").CombinedOutput()
	if err != nil {
		return readSubcmdCacheStale()
	}

	subcmds := parseClaudeSubcommands(string(out))
	if len(subcmds) > 0 {
		writeSubcmdCache(subcmds)
	}
	return subcmds
}

// isClaudeSubcommand checks whether the first positional argument in the
// claude args matches a real claude subcommand (queried from `claude --help`).
// If we can't determine the subcommands, we assume it's an interactive session.
func isClaudeSubcommand(claudeArgs []string) bool {
	for _, a := range claudeArgs {
		if strings.HasPrefix(a, "-") {
			continue
		}
		subcmds := getClaudeSubcommands()
		if subcmds == nil {
			return false // can't tell — default to session behavior
		}
		return subcmds[a]
	}
	return false
}

var resolveCmd = &cobra.Command{
	Use:           "resolve [path] [-- claude-args...]",
	Short:         "Resolve workspace for a path",
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
		// so commands are available on the very first launch of a new workspace.
		os.MkdirAll(result.SessionDir, 0755) //nolint:errcheck

		commands.Ensure()

		versionLine := "\033[90m↳ llmux " + DisplayVersion()
		if latest := update.CheckUpdateNoticeCached(DisplayVersion()); latest != "" {
			versionLine += " \033[33m(" + latest + " available — run llmux upgrade)\033[90m"
		}
		versionLine += "\033[0m\n"
		fmt.Fprint(os.Stderr, versionLine)
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
		// Check if the claude args indicate a subcommand (not an interactive session).
		// Session flags (--worktree, --enable-auto-mode) are only injected for
		// interactive sessions; subcommands like "mcp" and "config" get passed through.
		claudeArgs := args[1:] // everything after the path (passed via --)
		subcmd := isClaudeSubcommand(claudeArgs)

		// Line 3: worktree flag (always print to keep line-based protocol stable)
		if !subcmd && result.Worktree && isGitRepo(args[0]) {
			fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode enabled. Use --no-worktree to open claude normally.\033[0m\n")
			fmt.Print("\n--worktree")
		} else {
			if result.Worktree && !subcmd {
				fmt.Fprint(os.Stderr, "\033[90m↳ worktree mode skipped: not a git repository.\033[0m\n")
			}
			fmt.Print("\n")
		}
		// Line 4: auto mode flag (always print to keep line-based protocol stable)
		if !subcmd && result.AutoMode {
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
