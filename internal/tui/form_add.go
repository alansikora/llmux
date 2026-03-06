package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

type addFormData struct {
	FolderPath         string
	Name               string
	APIKey             string
	DisableAttribution bool
	Confirm            bool
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func newAddForm(data *addFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Folder path").
				Placeholder("~/Projects/myapp").
				Value(&data.FolderPath).
				Validate(func(s string) error {
					expanded := expandPath(s)
					abs, err := filepath.Abs(expanded)
					if err != nil {
						return err
					}
					info, err := os.Stat(abs)
					if err != nil {
						return err
					}
					if !info.IsDir() {
						return os.ErrNotExist
					}
					return nil
				}),
			huh.NewInput().
				Title("Workspace name").
				Placeholder("(defaults to folder name)").
				Value(&data.Name),
			huh.NewInput().
				Title("Anthropic API key").
				Placeholder("(optional, uses existing env if empty)").
				EchoMode(huh.EchoModePassword).
				Value(&data.APIKey),
			huh.NewConfirm().
				Title("Disable commit/PR attributions?").
				Description("Removes \"Made with Claude Code\" from commits and PRs").
				Affirmative("Yes").
				Negative("No").
				Value(&data.DisableAttribution),
			huh.NewConfirm().
				Title("Add this workspace?").
				Affirmative("Yes").
				Negative("No").
				Value(&data.Confirm),
		),
	)
}
