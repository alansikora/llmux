package tui

import "github.com/charmbracelet/huh"

type generalOptionsFormData struct {
	ShortAlias bool
}

func newGeneralOptionsForm(data *generalOptionsFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable short alias?").
				Description("Also define \"c\" as a shorthand for \"claude\" (requires shell restart)").
				Affirmative("Yes").
				Negative("No").
				Value(&data.ShortAlias),
		),
	).WithKeyMap(formKeyMap())
}
