package tui

import (
	"fmt"

	"github.com/allskar/llmux/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// confirmLeft creates a left-aligned confirm field.
func confirmLeft() *huh.Confirm {
	return huh.NewConfirm().WithButtonAlignment(lipgloss.Left)
}

var modifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

func dirtyTitle(title string, dirty bool) string {
	if dirty {
		return modifiedStyle.Render(title + " (modified)")
	}
	return title
}

// formKeyMap returns a keymap with up/down arrow support for navigating between fields.
func formKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Confirm.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Input.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Input.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Note.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Note.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Select.Next = key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "next"))
	km.Select.Prev = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back"))
	return km
}

// Unified options form data for both profile and project levels.
// Profile uses "enabled"/"disabled"; project adds "inherit".
type optionsFormData struct {
	Profile            string // project-level only: auth/credential profile name
	Worktree           string // "inherit", "enabled", "disabled"
	DisableAttribution string // "inherit", "enabled", "disabled"
}

// profileDefaults holds the profile-level defaults shown in "Inherit" labels.
type profileDefaults struct {
	Worktree           bool
	DisableAttribution bool
}

func selectOptions(label string, allowInherit bool, profileDefault *bool) []huh.Option[string] {
	var opts []huh.Option[string]
	if allowInherit {
		inheritLabel := label
		if profileDefault != nil {
			if *profileDefault {
				inheritLabel += " (currently: enabled)"
			} else {
				inheritLabel += " (currently: disabled)"
			}
		}
		opts = append(opts, huh.NewOption(inheritLabel, "inherit"))
	}
	opts = append(opts,
		huh.NewOption("Enabled", "enabled"),
		huh.NewOption("Disabled", "disabled"),
	)
	return opts
}

// newOptionsForm builds the unified options form. When profiles is non-empty,
// a profile picker is prepended and attribution is hidden (project-level edits;
// attribution is stored per-profile in session settings, not per-project).
// Otherwise the form shows attribution + worktree (profile-level edits).
func newOptionsForm(data *optionsFormData, orig optionsFormData, allowInherit bool, defaults *profileDefaults, profiles []config.Profile) *huh.Form {
	projectMode := len(profiles) > 0

	var fields []huh.Field

	if projectMode {
		profileOptions := make([]huh.Option[string], len(profiles))
		for i, pf := range profiles {
			label := pf.Name
			authInfo := config.GetAuthInfo(pf.Name)
			if authInfo.Authenticated {
				label = fmt.Sprintf("%s (%s)", pf.Name, authInfo.Email)
			}
			profileOptions[i] = huh.NewOption(label, pf.Name)
		}
		fields = append(fields,
			huh.NewSelect[string]().
				TitleFunc(func() string {
					return dirtyTitle("Profile", data.Profile != orig.Profile)
				}, &data.Profile).
				Description("Auth/credential profile for this project").
				Options(profileOptions...).
				Value(&data.Profile),
		)
	} else {
		var attrDefault *bool
		if defaults != nil {
			attrDefault = &defaults.DisableAttribution
		}
		fields = append(fields,
			huh.NewSelect[string]().
				TitleFunc(func() string {
					return dirtyTitle("Disable commit/PR attributions?", data.DisableAttribution != orig.DisableAttribution)
				}, &data.DisableAttribution).
				Description("Removes \"Made with Claude Code\" from commits and PRs").
				Options(selectOptions("Inherit from profile", allowInherit, attrDefault)...).
				Value(&data.DisableAttribution),
		)
	}

	worktreeSelect := huh.NewSelect[string]().
		TitleFunc(func() string {
			return dirtyTitle("Always use worktree?", data.Worktree != orig.Worktree)
		}, &data.Worktree).
		Description("Runs claude --worktree by default (bypass with --no-worktree)").
		Value(&data.Worktree)

	if projectMode {
		// Rebuild the "inherit" label whenever the profile picker changes so
		// the "currently: enabled/disabled" hint reflects the selected profile.
		worktreeSelect = worktreeSelect.OptionsFunc(func() []huh.Option[string] {
			var wd *bool
			for i := range profiles {
				if profiles[i].Name == data.Profile {
					v := profiles[i].Worktree
					wd = &v
					break
				}
			}
			return selectOptions("Inherit from profile", allowInherit, wd)
		}, &data.Profile)
	} else {
		var worktreeDefault *bool
		if defaults != nil {
			worktreeDefault = &defaults.Worktree
		}
		worktreeSelect = worktreeSelect.Options(selectOptions("Inherit from profile", allowInherit, worktreeDefault)...)
	}
	fields = append(fields, worktreeSelect)

	return huh.NewForm(huh.NewGroup(fields...)).WithKeyMap(formKeyMap())
}
