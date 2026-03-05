package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

type deleteFormData struct {
	Confirm bool
}

func newDeleteForm(name string, data *deleteFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Delete workspace %q?", name)).
				Description("This removes the workspace config. The session directory is kept.").
				Affirmative("Yes, delete").
				Negative("Cancel").
				Value(&data.Confirm),
		),
	)
}
