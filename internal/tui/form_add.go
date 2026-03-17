package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

// Workspace add form: name + optional API key (no path)
type wsAddFormData struct {
	Name   string
	APIKey string
}

func newWsAddForm(data *wsAddFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Workspace name").
				Placeholder("my-workspace").
				Value(&data.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return os.ErrInvalid
					}
					return nil
				}),
			huh.NewInput().
				Title("Anthropic API key").
				Placeholder("(optional, uses existing env if empty)").
				EchoMode(huh.EchoModePassword).
				Value(&data.APIKey),
		),
	).WithKeyMap(formKeyMap())
}

// Project add form: path only (workspace is implicit from context)
type projAddFormData struct {
	FolderPath string
}

func newProjAddForm(data *projAddFormData) *huh.Form {
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
		),
	).WithKeyMap(formKeyMap())
}
