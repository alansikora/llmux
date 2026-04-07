package tui

import "github.com/charmbracelet/huh"

type generalOptionsFormData struct {
	ShortAlias  bool
	ApplyMarker bool
	AutoMode    bool
	StatusLine  bool
	EffortLevel string
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
			confirmLeft().
				TitleFunc(func() string {
					return dirtyTitle("Enable auto mode?", data.AutoMode != orig.AutoMode)
				}, &data.AutoMode).
				Description("Enable auto mode in Claude Code, allowing it to run without confirmation prompts.").
				Affirmative("Enabled").
				Negative("Disabled").
				Value(&data.AutoMode),
			confirmLeft().
				TitleFunc(func() string {
					return dirtyTitle("Enable status line?", data.StatusLine != orig.StatusLine)
				}, &data.StatusLine).
				Description("Add ccstatusline to all workspace settings, showing model, context, and git info in Claude Code.").
				Affirmative("Enabled").
				Negative("Disabled").
				Value(&data.StatusLine),
			huh.NewSelect[string]().
				TitleFunc(func() string {
					return dirtyTitle("Default effort level", data.EffortLevel != orig.EffortLevel)
				}, &data.EffortLevel).
				Description("Set the default effort level for Claude Code sessions.").
				Options(
					huh.NewOption("Default (no flag)", ""),
					huh.NewOption("Min", "min"),
					huh.NewOption("Low", "low"),
					huh.NewOption("Medium", "medium"),
					huh.NewOption("High", "high"),
					huh.NewOption("Max (Opus 4.6 only)", "max"),
				).
				Value(&data.EffortLevel),
		),
	).WithKeyMap(formKeyMap())
}
