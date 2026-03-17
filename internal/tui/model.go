package tui

import (
	"fmt"
	"path/filepath"

	"github.com/allskar/llmux/internal/config"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateWorkspaceList    state = iota
	stateWorkspaceAdding        // Add workspace (name + API key)
	stateWorkspaceOptions       // Edit workspace settings
	stateWorkspaceDeleting      // Delete workspace confirmation
	stateProjectList            // List projects for a workspace
	stateProjectAdding          // Add project to workspace
	stateProjectOptions         // Edit project overrides
	stateSessions               // Worktree sessions (per project)
	stateGeneralOptions         // Global config options
)

type Model struct {
	cfg     *config.Config
	version string
	state   state

	list   list.Model // workspace list (main view)
	width  int
	height int

	// Workspace add form
	wsAddForm *huh.Form
	wsAddData wsAddFormData

	// Unified options form (used for both workspace and project)
	optionsForm   *huh.Form
	optionsData   optionsFormData
	wsOptionsTarget   string // workspace name (when editing workspace)
	projOptionsTarget string // project path (when editing project)

	// Workspace delete
	deleteForm   *huh.Form
	deleteData   deleteFormData
	deleteTarget string

	// Project list (per workspace)
	projectList    list.Model
	projectsTarget string // workspace name

	// Project add form
	projAddForm *huh.Form
	projAddData projAddFormData

	// Sessions view
	sessionsList   list.Model
	sessionsTarget string
	sessionsPath   string
	sessionsStatus string

	// Sessions loading
	sessionsLoading bool
	loadingProject  string
	spinner         spinner.Model

	// General options form
	generalOptionsForm *huh.Form
	generalOptionsData generalOptionsFormData
}

func NewModel(cfg *config.Config, version string) *Model {
	return &Model{
		cfg:     cfg,
		version: version,
		state:   stateWorkspaceList,
	}
}

func (m *Model) Init() tea.Cmd {
	m.list = buildWorkspaceList(m.cfg, m.version, 80, 20)
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Dot))
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.sessionsLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case sessionsLoadedMsg:
		m.sessionsLoading = false
		m.loadingProject = ""
		h, v := appStyle.GetFrameSize()
		m.sessionsList = buildSessionsList(msg.sessions, msg.applied, m.width-h, m.height-v-7)
		m.sessionsStatus = ""
		if msg.wsPath != "" {
			m.sessionsPath = msg.wsPath
		}
		m.sessionsTarget = msg.target
		m.state = stateSessions
		return m, nil
	case applyResultMsg:
		if msg.err != nil {
			m.sessionsStatus = fmt.Sprintf("error: %v", msg.err)
			return m, nil
		}
		m.sessionsStatus = fmt.Sprintf("applied %s", msg.session)
		return m, loadSessionsCmd(m.sessionsPath, m.sessionsTarget)
	case unapplyResultMsg:
		if msg.err != nil {
			m.sessionsStatus = fmt.Sprintf("error: %v", msg.err)
			return m, nil
		}
		m.sessionsStatus = "unapplied"
		return m, loadSessionsCmd(m.sessionsPath, m.sessionsTarget)
	case deleteResultMsg:
		if msg.err != nil {
			m.sessionsStatus = fmt.Sprintf("error: %v", msg.err)
			return m, nil
		}
		m.sessionsStatus = fmt.Sprintf("deleted %s", msg.session)
		return m, loadSessionsCmd(m.sessionsPath, m.sessionsTarget)
	case clipboardResultMsg:
		if msg.err != nil {
			m.sessionsStatus = fmt.Sprintf("copy failed: %v", msg.err)
		} else {
			m.sessionsStatus = fmt.Sprintf("copied path: %s", msg.path)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-7)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "esc" {
			switch m.state {
			case stateWorkspaceList:
				// Don't handle esc at top level
			case stateProjectList:
				m.state = stateWorkspaceList
				m.refreshWorkspaceList()
				return m, nil
			case stateSessions:
				// Go back to project list
				m.sessionsLoading = false
				projects := m.cfg.ProjectsForWorkspace(m.projectsTarget)
				h, v := appStyle.GetFrameSize()
				m.projectList = buildProjectList(projects, m.projectsTarget, m.width-h, m.height-v-7)
				m.state = stateProjectList
				return m, nil
			default:
				m.sessionsLoading = false
				// For forms, go back to appropriate parent
				switch m.state {
				case stateWorkspaceAdding, stateWorkspaceOptions, stateWorkspaceDeleting, stateGeneralOptions:
					m.state = stateWorkspaceList
					m.refreshWorkspaceList()
				case stateProjectAdding, stateProjectOptions:
					m.state = stateProjectList
					m.refreshProjectList()
				}
				return m, nil
			}
		}
	}

	switch m.state {
	case stateWorkspaceList:
		return updateWorkspaceList(m, msg)

	case stateWorkspaceAdding:
		form, cmd := m.wsAddForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.wsAddForm = f
		}
		if m.wsAddForm.State == huh.StateCompleted {
			m.applyWsAdd()
			m.state = stateWorkspaceList
			m.refreshWorkspaceList()
			return m, nil
		}
		if m.wsAddForm.State == huh.StateAborted {
			m.state = stateWorkspaceList
			return m, nil
		}
		return m, cmd

	case stateWorkspaceOptions:
		form, cmd := m.optionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.optionsForm = f
		}
		if m.optionsForm.State == huh.StateCompleted {
			m.applyWsOptions(m.wsOptionsTarget)
			m.state = stateWorkspaceList
			m.refreshWorkspaceList()
			return m, nil
		}
		if m.optionsForm.State == huh.StateAborted {
			m.state = stateWorkspaceList
			return m, nil
		}
		return m, cmd

	case stateWorkspaceDeleting:
		form, cmd := m.deleteForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.deleteForm = f
		}
		if m.deleteForm.State == huh.StateCompleted {
			if m.deleteData.Confirm {
				m.cfg.RemoveWorkspace(m.deleteTarget)
				config.Save(m.cfg)
			}
			m.state = stateWorkspaceList
			m.refreshWorkspaceList()
			return m, nil
		}
		if m.deleteForm.State == huh.StateAborted {
			m.state = stateWorkspaceList
			return m, nil
		}
		return m, cmd

	case stateProjectList:
		return updateProjectList(m, msg)

	case stateProjectAdding:
		form, cmd := m.projAddForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.projAddForm = f
		}
		if m.projAddForm.State == huh.StateCompleted {
			m.applyProjAdd()
			m.state = stateProjectList
			m.refreshProjectList()
			return m, nil
		}
		if m.projAddForm.State == huh.StateAborted {
			m.state = stateProjectList
			m.refreshProjectList()
			return m, nil
		}
		return m, cmd

	case stateProjectOptions:
		form, cmd := m.optionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.optionsForm = f
		}
		if m.optionsForm.State == huh.StateCompleted {
			m.applyProjOptions()
			m.state = stateProjectList
			m.refreshProjectList()
			return m, nil
		}
		if m.optionsForm.State == huh.StateAborted {
			m.state = stateProjectList
			m.refreshProjectList()
			return m, nil
		}
		return m, cmd

	case stateSessions:
		return updateSessions(m, msg)

	case stateGeneralOptions:
		form, cmd := m.generalOptionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.generalOptionsForm = f
		}
		if m.generalOptionsForm.State == huh.StateCompleted {
			m.cfg.ShortAlias = m.generalOptionsData.ShortAlias
			m.cfg.ApplyMarker = m.generalOptionsData.ApplyMarker
			m.cfg.AutoMode = m.generalOptionsData.AutoMode
			config.Save(m.cfg)
			m.state = stateWorkspaceList
			m.refreshWorkspaceList()
			return m, nil
		}
		if m.generalOptionsForm.State == huh.StateAborted {
			m.state = stateWorkspaceList
			return m, nil
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	var content string

	switch m.state {
	case stateWorkspaceList:
		header := logoStyle.Render(logo) + "  " + versionStyle.Render(m.version) + "\n\n"
		content = header + m.list.View()
	case stateWorkspaceAdding:
		content = titleStyle.Render("Add Workspace") + "\n\n" + m.wsAddForm.View()
	case stateWorkspaceOptions:
		content = titleStyle.Render("Options: "+m.wsOptionsTarget) + "\n\n" + m.optionsForm.View() + "\n" + hintStyle.Render("enter to save · esc to cancel")
	case stateWorkspaceDeleting:
		content = m.deleteForm.View()
	case stateProjectList:
		content = m.projectList.View()
	case stateProjectAdding:
		content = titleStyle.Render("Add Project to "+m.projectsTarget) + "\n\n" + m.projAddForm.View()
	case stateProjectOptions:
		content = titleStyle.Render("Project Options: "+filepath.Base(m.projOptionsTarget)) + "\n\n" + m.optionsForm.View() + "\n" + hintStyle.Render("enter to save · esc to cancel")
	case stateSessions:
		status := ""
		if m.sessionsStatus != "" {
			status = "\n" + statusBarStyle.Render(m.sessionsStatus)
		}
		content = m.sessionsList.View() + status
	case stateGeneralOptions:
		content = titleStyle.Render("General Options") + "\n\n" + m.generalOptionsForm.View() + "\n" + hintStyle.Render("enter to save · esc to cancel")
	}

	return appStyle.Render(lipgloss.Place(
		m.width, m.height,
		lipgloss.Left, lipgloss.Top,
		content,
	))
}

// --- Apply helpers ---

func (m *Model) applyWsAdd() {
	name := m.wsAddData.Name
	if err := m.cfg.AddWorkspace(name); err != nil {
		return
	}
	if m.wsAddData.APIKey != "" {
		for i := range m.cfg.Workspaces {
			if m.cfg.Workspaces[i].Name == name {
				m.cfg.Workspaces[i].APIKey = m.wsAddData.APIKey
				break
			}
		}
	}
	config.Save(m.cfg)
}

func (m *Model) applyWsOptions(name string) {
	// Update worktree setting
	for i := range m.cfg.Workspaces {
		if m.cfg.Workspaces[i].Name == name {
			m.cfg.Workspaces[i].Worktree = m.optionsData.Worktree == "enabled"
			break
		}
	}
	config.Save(m.cfg)

	// Update session settings (attribution)
	settings := config.ReadSessionSettings(name)
	if settings == nil {
		settings = map[string]any{}
	}
	if m.optionsData.DisableAttribution == "enabled" {
		settings["attribution"] = map[string]string{
			"commit": "",
			"pr":     "",
		}
	} else {
		delete(settings, "attribution")
	}
	config.WriteSessionSettings(name, settings)
}

func (m *Model) applyProjAdd() {
	path := expandPath(m.projAddData.FolderPath)
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	if err := m.cfg.AddProject(abs, m.projectsTarget); err != nil {
		return
	}
	config.Save(m.cfg)
}

func (m *Model) applyProjOptions() {
	for i := range m.cfg.Projects {
		if m.cfg.Projects[i].Path == m.projOptionsTarget {
			switch m.optionsData.Worktree {
			case "inherit":
				m.cfg.Projects[i].Overrides.Worktree = nil
			case "enabled":
				v := true
				m.cfg.Projects[i].Overrides.Worktree = &v
			case "disabled":
				v := false
				m.cfg.Projects[i].Overrides.Worktree = &v
			}
			break
		}
	}
	config.Save(m.cfg)
}

// --- Refresh helpers ---

func (m *Model) refreshWorkspaceList() {
	items := make([]list.Item, len(m.cfg.Workspaces))
	for i, ws := range m.cfg.Workspaces {
		items[i] = workspaceItem{
			name:         ws.Name,
			authInfo:     config.GetAuthInfo(ws.Name),
			isDefault:    ws.Name == m.cfg.DefaultWorkspace,
			projectCount: len(m.cfg.ProjectsForWorkspace(ws.Name)),
		}
	}
	m.list.SetItems(items)
}

func (m *Model) refreshProjectList() {
	projects := m.cfg.ProjectsForWorkspace(m.projectsTarget)
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{
			path:      p.Path,
			workspace: p.Workspace,
			overrides: p.Overrides,
		}
	}
	m.projectList.SetItems(items)
}
