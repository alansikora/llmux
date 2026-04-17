package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/allskar/llmux/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Profile list item (secondary view: profile management) ---

type profileItem struct {
	name         string
	authInfo     config.AuthInfo
	isDefault    bool
	projectCount int
	loading      string
}

func (p profileItem) Title() string {
	prefix := unauthStyle.Render("[no auth]")
	if p.authInfo.Authenticated {
		prefix = authStyle.Render("[" + p.authInfo.Email + "]")
	}
	if p.loading != "" {
		return fmt.Sprintf("%s %s %s", prefix, p.name, p.loading)
	}
	star := " "
	if p.isDefault {
		star = "★"
	}
	return fmt.Sprintf("%s %s %s", prefix, p.name, star)
}

func (p profileItem) Description() string {
	if p.projectCount == 1 {
		return "1 project"
	}
	return fmt.Sprintf("%d projects", p.projectCount)
}

func (p profileItem) FilterValue() string { return p.name }

// --- Project list item (main view) ---

type projectItem struct {
	name      string
	path      string
	profile   string
	authInfo  config.AuthInfo
	overrides config.ProjectOverrides
}

func (p projectItem) Title() string {
	prefix := unauthStyle.Render("[no auth]")
	if p.authInfo.Authenticated {
		prefix = authStyle.Render("[" + p.authInfo.Email + "]")
	}
	return fmt.Sprintf("%s %s", prefix, p.name)
}

func (p projectItem) Description() string {
	desc := fmt.Sprintf("%s · %s", p.path, p.profile)
	if p.overrides.Worktree != nil {
		if *p.overrides.Worktree {
			desc += " [worktree: on]"
		} else {
			desc += " [worktree: off]"
		}
	}
	return desc
}

func (p projectItem) FilterValue() string { return p.name }

// --- Item builders ---

func profileItems(cfg *config.Config) []list.Item {
	items := make([]list.Item, len(cfg.Profiles))
	for i, pf := range cfg.Profiles {
		items[i] = profileItem{
			name:         pf.Name,
			authInfo:     config.GetAuthInfo(pf.Name),
			isDefault:    pf.Name == cfg.DefaultProfile,
			projectCount: len(cfg.ProjectsForProfile(pf.Name)),
		}
	}
	return items
}

func projectItems(cfg *config.Config) []list.Item {
	sorted := make([]config.Project, len(cfg.Projects))
	copy(sorted, cfg.Projects)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(filepath.Base(sorted[i].Path)) < strings.ToLower(filepath.Base(sorted[j].Path))
	})
	items := make([]list.Item, len(sorted))
	for i, p := range sorted {
		items[i] = projectItem{
			name:      filepath.Base(p.Path),
			path:      p.Path,
			profile:   p.Profile,
			authInfo:  config.GetAuthInfo(p.Profile),
			overrides: p.Overrides,
		}
	}
	return items
}

// --- List builders ---

// topBarHeight is the vertical space reserved for the top bar header
// (1 line for breadcrumb/version + 1 blank separator line).
const topBarHeight = 2

// buildProjectList is the main view: a flat alphabetical list of all projects.
func buildProjectList(cfg *config.Config, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(projectItems(cfg), delegate, width, height)
	l.Title = "Projects"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "sessions")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
			key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "profiles")),
			key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "general options")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	return l
}

// buildProfileList is the secondary view: manage profiles (add/rename/edit/delete/set default).
func buildProfileList(cfg *config.Config, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(profileItems(cfg), delegate, width, height)
	l.Title = "Profiles"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter", "e"), key.WithHelp("enter", "edit")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
			key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "set default")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	return l
}

// --- Project list update (main view) ---

func updateProjectList(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a":
			if len(m.cfg.Profiles) == 0 {
				m.statusMsg = "no profiles configured — press 'p' to create one first"
				return m, nil
			}
			m.state = stateProjectAdding
			defaultProfile := m.cfg.DefaultProfile
			if defaultProfile == "" {
				defaultProfile = m.cfg.Profiles[0].Name
			}
			m.projAddData = projAddFormData{Profile: defaultProfile}
			m.projAddForm = newProjAddForm(&m.projAddData, m.cfg.Profiles)
			return m, m.projAddForm.Init()
		case "enter":
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				m.sessionsLoading = true
				m.loadingProject = item.path
				return m, tea.Batch(loadSessionsCmd(item.path, item.name), m.spinner.Tick)
			}
		case "e":
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				// Determine profile defaults for "inherit" labels
				var pf *config.Profile
				for i := range m.cfg.Profiles {
					if m.cfg.Profiles[i].Name == item.profile {
						pf = &m.cfg.Profiles[i]
						break
					}
				}
				defaults := &profileDefaults{}
				if pf != nil {
					defaults.Worktree = pf.Worktree
					defaults.DisableAttribution = config.IsAttributionDisabled(pf.Name)
				}

				worktreeVal := "inherit"
				if item.overrides.Worktree != nil {
					if *item.overrides.Worktree {
						worktreeVal = "enabled"
					} else {
						worktreeVal = "disabled"
					}
				}
				m.projOptionsTarget = item.path
				m.optionsData = optionsFormData{
					Profile:            item.profile,
					Worktree:           worktreeVal,
					DisableAttribution: "inherit",
				}
				m.optionsForm = newOptionsForm(&m.optionsData, m.optionsData, true, defaults, m.cfg.Profiles)
				m.state = stateProjectOptions
				return m, m.optionsForm.Init()
			}
		case "d", "x":
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				if err := m.cfg.RemoveProject(item.path); err != nil {
					m.statusMsg = fmt.Sprintf("remove error: %v", err)
					return m, nil
				}
				if err := config.Save(m.cfg); err != nil {
					m.statusMsg = fmt.Sprintf("save error: %v", err)
					return m, nil
				}
				m.statusMsg = ""
				m.refreshProjectList()
				return m, nil
			}
		case "p":
			h, v := appStyle.GetFrameSize()
			m.profileList = buildProfileList(m.cfg, m.width-h, m.height-v-topBarHeight)
			m.profileListReady = true
			m.state = stateProfileList
			return m, nil
		case "o":
			m.generalOptionsData = generalOptionsFormData{
				ShortAlias:         m.cfg.ShortAlias,
				ApplyMarker:        m.cfg.ApplyMarker,
				AutoMode:           m.cfg.AutoMode,
				StatusLine:         m.cfg.StatusLine,
				AutoDefaultProfile: m.cfg.AutoDefaultProfile,
			}
			m.generalOptionsForm = newGeneralOptionsForm(&m.generalOptionsData, m.generalOptionsData)
			m.state = stateGeneralOptions
			return m, m.generalOptionsForm.Init()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// --- Profile list update (secondary view) ---

func updateProfileList(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.profileList.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a":
			m.state = stateProfileAdding
			defaultName := ""
			if cwd, err := os.Getwd(); err == nil {
				defaultName = filepath.Base(cwd)
			}
			m.profileAddData = profileAddFormData{Name: defaultName}
			m.profileAddForm = newProfileAddForm(&m.profileAddData)
			return m, m.profileAddForm.Init()
		case "r":
			if item, ok := m.profileList.SelectedItem().(profileItem); ok {
				m.profileRenameTarget = item.name
				m.profileRenameData = profileRenameFormData{Name: item.name}
				m.profileRenameForm = newProfileRenameForm(&m.profileRenameData)
				m.state = stateProfileRenaming
				return m, m.profileRenameForm.Init()
			}
		case "e", "enter":
			if item, ok := m.profileList.SelectedItem().(profileItem); ok {
				attrVal := "disabled"
				if config.IsAttributionDisabled(item.name) {
					attrVal = "enabled"
				}
				worktreeVal := "disabled"
				for _, pf := range m.cfg.Profiles {
					if pf.Name == item.name {
						if pf.Worktree {
							worktreeVal = "enabled"
						}
						break
					}
				}
				m.profileOptionsTarget = item.name
				m.optionsData = optionsFormData{
					DisableAttribution: attrVal,
					Worktree:           worktreeVal,
				}
				m.optionsForm = newOptionsForm(&m.optionsData, m.optionsData, false, nil, nil)
				m.state = stateProfileOptions
				return m, m.optionsForm.Init()
			}
		case "s":
			if item, ok := m.profileList.SelectedItem().(profileItem); ok {
				if m.cfg.DefaultProfile == item.name {
					m.cfg.SetDefaultProfile("")
				} else {
					m.cfg.SetDefaultProfile(item.name)
				}
				if err := config.Save(m.cfg); err != nil {
					m.statusMsg = fmt.Sprintf("save error: %v", err)
					return m, nil
				}
				m.statusMsg = ""
				m.refreshProfileList()
				return m, nil
			}
		case "d", "x":
			if item, ok := m.profileList.SelectedItem().(profileItem); ok {
				m.state = stateProfileDeleting
				m.deleteTarget = item.name
				m.deleteData = deleteFormData{}
				m.deleteForm = newDeleteForm("profile", item.name, &m.deleteData)
				return m, m.deleteForm.Init()
			}
		}
	}

	var cmd tea.Cmd
	m.profileList, cmd = m.profileList.Update(msg)
	return m, cmd
}
