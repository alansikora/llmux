package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/worktree"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [session]",
	Short: "Apply worktree session changes to the main working tree",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		var sessionName string
		if len(args) > 0 {
			sessionName = args[0]
		} else {
			detected, err := worktree.DetectCurrentSession(cwd)
			if err != nil {
				return fmt.Errorf("no session name provided and not inside a worktree: %w", err)
			}
			sessionName = detected
		}

		var profileArgs []string
		if applyProfile != "" {
			profileArgs = []string{applyProfile}
		}
		_, _, err = resolveProfile(cfg, profileArgs)
		if err != nil {
			return err
		}

		// Prefer git repo root for finding sessions
		sessionsPath := worktree.ResolveSessionsPath(cwd)

		if err := worktree.Apply(sessionsPath, sessionName, cfg.ApplyMarker); err != nil {
			return err
		}

		fmt.Printf("Applied session %q to %s\n", sessionName, sessionsPath)
		return nil
	},
}

var applyProfile string

func init() {
	applyCmd.Flags().StringVarP(&applyProfile, "profile", "p", "", "profile name")
	rootCmd.AddCommand(applyCmd)
}
