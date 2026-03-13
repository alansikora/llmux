package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/worktree"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <session-name-or-branch>",
	Short: "Resume a worktree session by name or branch",
	Long:  "Finds the worktree session, sets up the workspace environment, and launches claude --continue from the worktree directory.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ws, err := resolveWorkspace(cfg, nil)
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		sessionsPath := worktree.ResolveSessionsPath(cwd)

		sessions, err := worktree.ListSessions(sessionsPath)
		if err != nil {
			return fmt.Errorf("listing sessions: %w", err)
		}

		var match *worktree.Session
		for i := range sessions {
			if sessions[i].Name == query || sessions[i].Branch == query {
				match = &sessions[i]
				break
			}
		}
		if match == nil {
			return fmt.Errorf("no session found matching %q", query)
		}

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH: %w", err)
		}

		sessionDir := config.SessionDir(ws.Name)
		ensureCommandsSymlink(sessionDir)

		env := os.Environ()
		env = setEnv(env, "CLAUDE_CONFIG_DIR", sessionDir)
		if ws.APIKey != "" {
			env = setEnv(env, "ANTHROPIC_API_KEY", ws.APIKey)
		}

		fmt.Fprintf(os.Stderr, "\033[90m↳ resuming session %s in %s\033[0m\n", match.Name, match.Path)

		if err := syscall.Chdir(match.Path); err != nil {
			return fmt.Errorf("changing to worktree directory: %w", err)
		}

		return syscall.Exec(claudePath, []string{"claude", "--continue"}, env)
	},
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
