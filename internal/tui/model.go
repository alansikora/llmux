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
	stateList state = iota
	stateAdding
	stateAddOptions
	stateOptions
	stateDeleting
	stateSessions
	stateGeneralOptions
)

type Model struct {
	cfg     *config.Config
	version string
	state   state

	list   list.Model
	width  int
	height int

	// Add form
	addForm *huh.Form
	addData addFormData

	// Options form (shared by add flow and standalone options)
	optionsForm   *huh.Form
	optionsData   optionsFormData
	optionsTarget string

	// Delete form
	deleteForm   *huh.Form
	deleteData   deleteFormData
	deleteTarget string

	// Sessions view
	sessionsList   list.Model
	sessionsTarget string
	sessionsPath   string
	sessionsStatus string

	// Sessions loading (inline spinner on workspace list)
	sessionsLoading  bool
	loadingWorkspace string
	spinner          spinner.Model

	// General options form
	generalOptionsForm *huh.Form
	generalOptionsData generalOptionsFormData
}

func NewModel(cfg *config.Config, version string) *Model {
	return &Model{
		cfg:     cfg,
		version: version,
		state:   stateList,
	}
}

func (m *Model) Init() tea.Cmd {
	m.list = buildList(m.cfg, m.version, 80, 20)
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Dot))
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.sessionsLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			m.updateLoadingItem()
			return m, cmd
		}
		return m, nil
	case sessionsLoadedMsg:
		m.sessionsLoading = false
		m.loadingWorkspace = ""
		m.refreshList()
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
		if msg.String() == "esc" && m.state != stateList {
			m.sessionsLoading = false
			m.state = stateList
			m.refreshList()
			return m, nil
		}
	}

	switch m.state {
	case stateList:
		return updateList(m, msg)

	case stateAdding:
		form, cmd := m.addForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.addForm = f
		}

		if m.addForm.State == huh.StateCompleted {
			// Transition to options form
			name := m.resolveAddName()
			m.optionsTarget = name
			m.optionsData = optionsFormData{}
			m.optionsForm = newOptionsForm(&m.optionsData, m.optionsData)
			m.state = stateAddOptions
			return m, m.optionsForm.Init()
		}
		if m.addForm.State == huh.StateAborted {
			m.state = stateList
			return m, nil
		}
		return m, cmd

	case stateAddOptions:
		form, cmd := m.optionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.optionsForm = f
		}

		if m.optionsForm.State == huh.StateCompleted {
			m.applyAdd()
			m.applyOptions(m.optionsTarget)
			m.state = stateList
			m.refreshList()
			return m, nil
		}
		if m.optionsForm.State == huh.StateAborted {
			m.state = stateList
			return m, nil
		}
		return m, cmd

	case stateOptions:
		form, cmd := m.optionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.optionsForm = f
		}

		if m.optionsForm.State == huh.StateCompleted {
			m.applyOptions(m.optionsTarget)
			m.state = stateList
			m.refreshList()
			return m, nil
		}
		if m.optionsForm.State == huh.StateAborted {
			m.state = stateList
			return m, nil
		}
		return m, cmd

	case stateDeleting:
		form, cmd := m.deleteForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.deleteForm = f
		}

		if m.deleteForm.State == huh.StateCompleted {
			if m.deleteData.Confirm {
				m.cfg.Remove(m.deleteTarget)
				config.Save(m.cfg)
			}
			m.state = stateList
			m.refreshList()
			return m, nil
		}
		if m.deleteForm.State == huh.StateAborted {
			m.state = stateList
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
			config.Save(m.cfg)
			m.state = stateList
			m.refreshList()
			return m, nil
		}
		if m.generalOptionsForm.State == huh.StateAborted {
			m.state = stateList
			return m, nil
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	var content string

	switch m.state {
	case stateList:
		header := logoStyle.Render(logo) + "  " + versionStyle.Render(m.version) + "\n\n"
		content = header + m.list.View()
	case stateAdding:
		content = titleStyle.Render("Add Workspace") + "\n\n" + m.addForm.View()
	case stateAddOptions:
		content = titleStyle.Render("Workspace Options") + "\n\n" + m.optionsForm.View() + "\n" + hintStyle.Render("enter to save · esc to cancel")
	case stateOptions:
		content = titleStyle.Render("Options: "+m.optionsTarget) + "\n\n" + m.optionsForm.View() + "\n" + hintStyle.Render("enter to save · esc to cancel")
	case stateDeleting:
		content = m.deleteForm.View()
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

func (m *Model) resolveAddName() string {
	name := m.addData.Name
	if name == "" {
		path := expandPath(m.addData.FolderPath)
		abs, err := filepath.Abs(path)
		if err != nil {
			return ""
		}
		name = filepath.Base(abs)
	}
	return name
}

func (m *Model) applyAdd() {
	path := expandPath(m.addData.FolderPath)
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}

	name := m.addData.Name
	if name == "" {
		name = filepath.Base(abs)
	}

	if err := m.cfg.Add(name, abs); err != nil {
		return
	}

	if m.addData.APIKey != "" {
		for i := range m.cfg.Workspaces {
			if m.cfg.Workspaces[i].Name == name {
				m.cfg.Workspaces[i].APIKey = m.addData.APIKey
				break
			}
		}
	}

	config.Save(m.cfg)
}

func (m *Model) applyOptions(name string) {
	// Update worktree setting in config
	for i := range m.cfg.Workspaces {
		if m.cfg.Workspaces[i].Name == name {
			m.cfg.Workspaces[i].Worktree = m.optionsData.AlwaysWorktree
			break
		}
	}
	config.Save(m.cfg)

	// Update session settings (attribution)
	settings := config.ReadSessionSettings(name)
	if settings == nil {
		settings = map[string]any{}
	}
	if m.optionsData.DisableAttribution {
		settings["attribution"] = map[string]string{
			"commit": "",
			"pr":     "",
		}
	} else {
		delete(settings, "attribution")
	}
	config.WriteSessionSettings(name, settings)
}

func (m *Model) updateLoadingItem() {
	items := m.list.Items()
	for i, item := range items {
		if ws, ok := item.(workspaceItem); ok {
			if ws.name == m.loadingWorkspace {
				ws.loading = m.spinner.View()
			} else {
				ws.loading = ""
			}
			items[i] = ws
		}
	}
	m.list.SetItems(items)
}

func (m *Model) refreshList() {
	items := make([]list.Item, len(m.cfg.Workspaces))
	for i, ws := range m.cfg.Workspaces {
		items[i] = workspaceItem{
			name:      ws.Name,
			path:      ws.Path,
			authInfo:  config.GetAuthInfo(ws.Name),
			isDefault: ws.Name == m.cfg.DefaultWorkspace,
		}
	}
	m.list.SetItems(items)
}
