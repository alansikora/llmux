package tui

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/allskar/llmux/internal/config"
)

// --- Workspace list item ---

type workspaceItem struct {
	name         string
	authInfo     config.AuthInfo
	isDefault    bool
	projectCount int
	loading      string
}

func (w workspaceItem) Title() string {
	if w.loading != "" {
		prefix := unauthStyle.Render("[no auth]")
		if w.authInfo.Authenticated {
			prefix = authStyle.Render("[" + w.authInfo.Email + "]")
		}
		return fmt.Sprintf("%s %s %s", prefix, w.name, w.loading)
	}

	prefix := unauthStyle.Render("[no auth]")
	if w.authInfo.Authenticated {
		prefix = authStyle.Render("[" + w.authInfo.Email + "]")
	}

	star := " "
	if w.isDefault {
		star = "★"
	}

	return fmt.Sprintf("%s %s %s", prefix, w.name, star)
}

func (w workspaceItem) Description() string {
	if w.projectCount == 1 {
		return "1 project"
	}
	return fmt.Sprintf("%d projects", w.projectCount)
}

func (w workspaceItem) FilterValue() string { return w.name }

// --- Project list item ---

type projectItem struct {
	path      string
	workspace string
	overrides config.ProjectOverrides
}

func (p projectItem) Title() string {
	return filepath.Base(p.path)
}

func (p projectItem) Description() string {
	desc := p.path
	if p.overrides.Worktree != nil {
		if *p.overrides.Worktree {
			desc += " [worktree: on]"
		} else {
			desc += " [worktree: off]"
		}
	}
	return desc
}

func (p projectItem) FilterValue() string { return filepath.Base(p.path) }

// --- List builders ---

const logo = ` _ _
| | |_ __ ___  _   ___  __
| | | '_ ` + "`" + ` _ \| | | \ \/ /
| | | | | | | | |_| |>  <
|_|_|_| |_| |_|\__,_/_/\_\`

func buildWorkspaceList(cfg *config.Config, version string, width, height int) list.Model {
	items := make([]list.Item, len(cfg.Workspaces))
	for i, ws := range cfg.Workspaces {
		items[i] = workspaceItem{
			name:         ws.Name,
			authInfo:     config.GetAuthInfo(ws.Name),
			isDefault:    ws.Name == cfg.DefaultWorkspace,
			projectCount: len(cfg.ProjectsForWorkspace(ws.Name)),
		}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, width, height)
	l.Title = "Workspaces"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "projects")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
			key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "set default")),
			key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "general options")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	return l
}

func buildProjectList(projects []config.Project, wsName string, width, height int) list.Model {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{
			path:      p.Path,
			workspace: p.Workspace,
			overrides: p.Overrides,
		}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, width, height)
	l.Title = fmt.Sprintf("Projects: %s", wsName)
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "sessions")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
			key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	return l
}

// --- Workspace list update ---

func updateWorkspaceList(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a":
			m.state = stateWorkspaceAdding
			m.wsAddData = wsAddFormData{}
			m.wsAddForm = newWsAddForm(&m.wsAddData)
			return m, m.wsAddForm.Init()
		case "enter":
			if item, ok := m.list.SelectedItem().(workspaceItem); ok {
				m.projectsTarget = item.name
				projects := m.cfg.ProjectsForWorkspace(item.name)
				h, v := appStyle.GetFrameSize()
				m.projectList = buildProjectList(projects, item.name, m.width-h, m.height-v-7)
				m.state = stateProjectList
				return m, nil
			}
		case "e":
			if item, ok := m.list.SelectedItem().(workspaceItem); ok {
				settings := config.ReadSessionSettings(item.name)
				attrVal := "disabled"
				if settings != nil {
					if _, ok := settings["attribution"]; ok {
						attrVal = "enabled"
					}
				}
				worktreeVal := "disabled"
				for _, ws := range m.cfg.Workspaces {
					if ws.Name == item.name {
						if ws.Worktree {
							worktreeVal = "enabled"
						}
						break
					}
				}
				m.wsOptionsTarget = item.name
				m.optionsData = optionsFormData{
					DisableAttribution: attrVal,
					Worktree:           worktreeVal,
				}
				m.optionsForm = newOptionsForm(&m.optionsData, m.optionsData, false, nil)
				m.state = stateWorkspaceOptions
				return m, m.optionsForm.Init()
			}
		case "s":
			if item, ok := m.list.SelectedItem().(workspaceItem); ok {
				if m.cfg.DefaultWorkspace == item.name {
					m.cfg.SetDefault("")
				} else {
					m.cfg.SetDefault(item.name)
				}
				config.Save(m.cfg)
				m.refreshWorkspaceList()
				return m, nil
			}
		case "o":
			m.generalOptionsData = generalOptionsFormData{
				ShortAlias:  m.cfg.ShortAlias,
				ApplyMarker: m.cfg.ApplyMarker,
				AutoMode:    m.cfg.AutoMode,
			}
			m.generalOptionsForm = newGeneralOptionsForm(&m.generalOptionsData, m.generalOptionsData)
			m.state = stateGeneralOptions
			return m, m.generalOptionsForm.Init()
		case "d", "x":
			if item, ok := m.list.SelectedItem().(workspaceItem); ok {
				m.state = stateWorkspaceDeleting
				m.deleteTarget = item.name
				m.deleteData = deleteFormData{}
				m.deleteForm = newDeleteForm(item.name, &m.deleteData)
				return m, m.deleteForm.Init()
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// --- Project list update ---

func updateProjectList(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.projectList.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a":
			m.state = stateProjectAdding
			m.projAddData = projAddFormData{}
			m.projAddForm = newProjAddForm(&m.projAddData)
			return m, m.projAddForm.Init()
		case "enter":
			if item, ok := m.projectList.SelectedItem().(projectItem); ok {
				m.sessionsLoading = true
				m.loadingProject = item.path
				return m, tea.Batch(loadSessionsCmd(item.path, m.projectsTarget), m.spinner.Tick)
			}
		case "e":
			if item, ok := m.projectList.SelectedItem().(projectItem); ok {
				// Determine workspace defaults for "inherit" labels
				var ws *config.Workspace
				for i := range m.cfg.Workspaces {
					if m.cfg.Workspaces[i].Name == m.projectsTarget {
						ws = &m.cfg.Workspaces[i]
						break
					}
				}
				defaults := &wsDefaults{}
				if ws != nil {
					defaults.Worktree = ws.Worktree
					settings := config.ReadSessionSettings(ws.Name)
					if settings != nil {
						if _, ok := settings["attribution"]; ok {
							defaults.DisableAttribution = true
						}
					}
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
					Worktree:           worktreeVal,
					DisableAttribution: "inherit",
				}
				m.optionsForm = newOptionsForm(&m.optionsData, m.optionsData, true, defaults)
				m.state = stateProjectOptions
				return m, m.optionsForm.Init()
			}
		case "d", "x":
			if item, ok := m.projectList.SelectedItem().(projectItem); ok {
				m.cfg.RemoveProject(item.path)
				config.Save(m.cfg)
				m.refreshProjectList()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.projectList, cmd = m.projectList.Update(msg)
	return m, cmd
}
