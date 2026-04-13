package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/allskar/llmux/internal/config"
	"github.com/charmbracelet/huh"
)

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

// Profile add form: name only
type profileAddFormData struct {
	Name string
}

func newProfileAddForm(data *profileAddFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Placeholder("my-profile").
				Value(&data.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return os.ErrInvalid
					}
					return nil
				}),
		),
	).WithKeyMap(formKeyMap())
}

// Profile rename form: new name only
type profileRenameFormData struct {
	Name string
}

func newProfileRenameForm(data *profileRenameFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("New profile name").
				Placeholder("new-profile-name").
				Value(&data.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return os.ErrInvalid
					}
					return nil
				}),
		),
	).WithKeyMap(formKeyMap())
}

// Project add form: folder path + profile picker
type projAddFormData struct {
	FolderPath string
	Profile    string
}

func newProjAddForm(data *projAddFormData, profiles []config.Profile) *huh.Form {
	options := make([]huh.Option[string], len(profiles))
	for i, pf := range profiles {
		label := pf.Name
		authInfo := config.GetAuthInfo(pf.Name)
		if authInfo.Authenticated {
			label = fmt.Sprintf("%s (%s)", pf.Name, authInfo.Email)
		}
		options[i] = huh.NewOption(label, pf.Name)
	}

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
			huh.NewSelect[string]().
				Title("Profile").
				Description("Which auth/credential profile to use for this project").
				Options(options...).
				Value(&data.Profile),
		),
	).WithKeyMap(formKeyMap())
}
