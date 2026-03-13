package tui

import "github.com/charmbracelet/huh"

type generalOptionsFormData struct {
	ShortAlias  bool
	ApplyMarker bool
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
			confirmLeft().
				TitleFunc(func() string {
					return dirtyTitle("Add workspace apply marker?", data.ApplyMarker != orig.ApplyMarker)
				}, &data.ApplyMarker).
				Description("When a worktree session is applied, create a .llmux-applied file in the workspace root.\nThis makes it visible in git status that changes from a worktree session are overlaid on your working tree.").
				Affirmative("Enabled").
				Negative("Disabled").
				Value(&data.ApplyMarker),
		),
	).WithKeyMap(formKeyMap())
}
