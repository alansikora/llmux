package tui

import "github.com/charmbracelet/huh"

type generalOptionsFormData struct {
	ShortAlias bool
}

func newGeneralOptionsForm(data *generalOptionsFormData, orig generalOptionsFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			confirmLeft().
				TitleFunc(func() string {
					return dirtyTitle("Enable short alias?", data.ShortAlias != orig.ShortAlias)
				}, &data.ShortAlias).
				Description("Also define \"c\" as a shorthand for \"claude\" (requires shell restart)").
				Affirmative("Enabled").
				Negative("Disabled").
				Value(&data.ShortAlias),
		),
	).WithKeyMap(formKeyMap())
}
