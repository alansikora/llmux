package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

type deleteFormData struct {
	Confirm bool
}

// newDeleteForm returns a confirmation form for deleting an entity.
// entityType is a label like "profile", "project", or "session".
func newDeleteForm(entityType, name string, data *deleteFormData) *huh.Form {
	var desc string
	switch entityType {
	case "profile":
		desc = "This removes the profile config, session directory (auth, history, settings), and all associated projects."
	case "session":
		desc = "This deletes the git worktree and removes its registration. Uncommitted changes in the worktree will be lost."
	default:
		desc = fmt.Sprintf("This removes the %s config.", entityType)
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Delete %s %q?", entityType, name)).
				Description(desc).
				Affirmative("Yes, delete").
				Negative("Cancel").
				Value(&data.Confirm),
		),
	).WithKeyMap(formKeyMap())
}
