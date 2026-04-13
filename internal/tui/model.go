package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/allskar/llmux/internal/config"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type state int

const (
	stateProjectList     state = iota // MAIN VIEW: flat alphabetical list of all projects
	stateProjectAdding                // Add project (path + profile picker)
	stateProjectOptions               // Edit project overrides
	stateSessions                     // Worktree sessions (per project)
	stateSessionDeleting              // Delete session confirmation (destructive)
	stateGeneralOptions               // Global config options
	stateProfileList                  // Profile management (secondary view)
	stateProfileAdding                // Add profile
	stateProfileRenaming              // Rename profile
	stateProfileOptions               // Edit profile settings
	stateProfileDeleting              // Delete profile confirmation
)

// updateCheckMsg is sent when the async update check completes.
type updateCheckMsg struct {
	latestVersion string
}

type Model struct {
	cfg           *config.Config
	version       string
	latestVersion string        // set if a newer version is available
	updateCh      <-chan string // receives result from async update check
	state         state

	list   list.Model // project list (main view)
	width  int
	height int

	// Profile add form
	profileAddForm *huh.Form
	profileAddData profileAddFormData

	// Profile rename form
	profileRenameForm   *huh.Form
	profileRenameData   profileRenameFormData
	profileRenameTarget string // original profile name

	// Unified options form (used for both profile and project)
	optionsForm          *huh.Form
	optionsData          optionsFormData
	profileOptionsTarget string // profile name (when editing profile)
	projOptionsTarget    string // project path (when editing project)

	// Delete confirmation (generic, used for profile + session)
	deleteForm          *huh.Form
	deleteData          deleteFormData
	deleteTarget        string
	deleteSessionWsPath string // workspace path when deleting a session

	// Profile list (secondary view)
	profileList list.Model

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

	// Status message shown in project list view
	statusMsg string
}

func NewModel(cfg *config.Config, version string, updateCh <-chan string) *Model {
	return &Model{
		cfg:      cfg,
		version:  version,
		updateCh: updateCh,
		state:    stateProjectList,
	}
}

func (m *Model) Init() tea.Cmd {
	m.list = buildProjectList(m.cfg, 80, 20)
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Dot))
	if m.updateCh != nil {
		return waitForUpdateCheck(m.updateCh)
	}
	return nil
}

func waitForUpdateCheck(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		return updateCheckMsg{latestVersion: <-ch}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateCheckMsg:
		m.latestVersion = msg.latestVersion
		return m, nil
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
		m.sessionsList = buildSessionsList(msg.sessions, msg.applied, msg.target, m.width-h, m.height-v-topBarHeight)
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
		m.list.SetSize(msg.Width-h, msg.Height-v-topBarHeight)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "esc" {
			switch m.state {
			case stateProjectList:
				// Top level: esc quits.
				return m, tea.Quit
			case stateSessions:
				m.sessionsLoading = false
				m.state = stateProjectList
				m.refreshProjectList()
				return m, nil
			case stateProfileList:
				m.state = stateProjectList
				m.refreshProjectList()
				return m, nil
			case stateSessionDeleting:
				m.state = stateSessions
				return m, nil
			default:
				m.sessionsLoading = false
				// For forms, go back to appropriate parent
				switch m.state {
				case stateProfileAdding, stateProfileRenaming, stateProfileOptions, stateProfileDeleting:
					m.state = stateProfileList
					m.refreshProfileList()
				case stateProjectAdding, stateProjectOptions, stateGeneralOptions:
					m.state = stateProjectList
					m.refreshProjectList()
				}
				return m, nil
			}
		}
	}

	switch m.state {
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

	case stateSessionDeleting:
		form, cmd := m.deleteForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.deleteForm = f
		}
		if m.deleteForm.State == huh.StateCompleted {
			if m.deleteData.Confirm {
				name := m.deleteTarget
				wsPath := m.deleteSessionWsPath
				m.sessionsStatus = fmt.Sprintf("deleting %s...", name)
				m.state = stateSessions
				return m, deleteSessionCmd(wsPath, name, false)
			}
			m.state = stateSessions
			return m, nil
		}
		if m.deleteForm.State == huh.StateAborted {
			m.state = stateSessions
			return m, nil
		}
		return m, cmd

	case stateGeneralOptions:
		form, cmd := m.generalOptionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.generalOptionsForm = f
		}
		if m.generalOptionsForm.State == huh.StateCompleted {
			m.cfg.ShortAlias = m.generalOptionsData.ShortAlias
			m.cfg.ApplyMarker = m.generalOptionsData.ApplyMarker
			m.cfg.AutoMode = m.generalOptionsData.AutoMode
			m.cfg.StatusLine = m.generalOptionsData.StatusLine
			config.Save(m.cfg)
			if err := config.SyncAllProfileSettings(m.cfg); err != nil {
				m.statusMsg = fmt.Sprintf("settings sync error: %v", err)
			} else {
				m.statusMsg = ""
			}
			m.state = stateProjectList
			m.refreshProjectList()
			return m, nil
		}
		if m.generalOptionsForm.State == huh.StateAborted {
			m.state = stateProjectList
			return m, nil
		}
		return m, cmd

	case stateProfileList:
		return updateProfileList(m, msg)

	case stateProfileAdding:
		form, cmd := m.profileAddForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.profileAddForm = f
		}
		if m.profileAddForm.State == huh.StateCompleted {
			m.applyProfileAdd()
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		if m.profileAddForm.State == huh.StateAborted {
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		return m, cmd

	case stateProfileRenaming:
		form, cmd := m.profileRenameForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.profileRenameForm = f
		}
		if m.profileRenameForm.State == huh.StateCompleted {
			m.applyProfileRename()
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		if m.profileRenameForm.State == huh.StateAborted {
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		return m, cmd

	case stateProfileOptions:
		form, cmd := m.optionsForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.optionsForm = f
		}
		if m.optionsForm.State == huh.StateCompleted {
			m.applyProfileOptions(m.profileOptionsTarget)
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		if m.optionsForm.State == huh.StateAborted {
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		return m, cmd

	case stateProfileDeleting:
		form, cmd := m.deleteForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.deleteForm = f
		}
		if m.deleteForm.State == huh.StateCompleted {
			if m.deleteData.Confirm {
				m.cfg.RemoveProfile(m.deleteTarget)
				if err := config.Save(m.cfg); err != nil {
					// Reload config from disk to restore the original state
					if restored, loadErr := config.Load(); loadErr == nil {
						m.cfg = restored
					} else {
						m.statusMsg = fmt.Sprintf("save error: %v; failed to reload config: %v", err, loadErr)
						m.state = stateProfileList
						m.refreshProfileList()
						return m, nil
					}
					m.statusMsg = fmt.Sprintf("save error: %v", err)
				} else if err := config.RemoveSessionDir(m.deleteTarget); err != nil {
					m.statusMsg = fmt.Sprintf("remove session dir error: %v", err)
				}
			}
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		if m.deleteForm.State == huh.StateAborted {
			m.state = stateProfileList
			m.refreshProfileList()
			return m, nil
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	// Top bar: breadcrumb on the left, llmux version (+ update notice) on the right.
	h, _ := appStyle.GetFrameSize()
	header := topBar(m.width-h, m.leftBar(), m.rightBar()) + "\n\n"

	var body string
	saveHint := "\n" + hintStyle.Render("enter to save · esc to cancel")

	switch m.state {
	case stateProjectList:
		body = m.list.View()
		if !m.sessionsLoading && m.statusMsg != "" {
			body += "\n" + statusBarStyle.Render(m.statusMsg)
		}
	case stateProjectAdding:
		body = m.projAddForm.View() + saveHint
	case stateProjectOptions:
		body = m.optionsForm.View() + saveHint
	case stateSessions:
		body = m.sessionsList.View()
		if m.sessionsStatus != "" {
			body += "\n" + statusBarStyle.Render(m.sessionsStatus)
		} else if m.statusMsg != "" {
			body += "\n" + statusBarStyle.Render(m.statusMsg)
		}
	case stateSessionDeleting:
		body = m.deleteForm.View()
	case stateGeneralOptions:
		body = m.generalOptionsForm.View() + saveHint
	case stateProfileList:
		body = m.profileList.View()
		if m.statusMsg != "" {
			body += "\n" + statusBarStyle.Render(m.statusMsg)
		}
	case stateProfileAdding:
		body = m.profileAddForm.View() + saveHint
	case stateProfileRenaming:
		body = m.profileRenameForm.View() + saveHint
	case stateProfileOptions:
		body = m.optionsForm.View() + saveHint
	case stateProfileDeleting:
		body = m.deleteForm.View()
	}

	return appStyle.Render(header + body)
}

// leftBar builds the location breadcrumb for the current state.
func (m *Model) leftBar() string {
	// While sessions are loading from the project list, preview the
	// destination breadcrumb with a spinner suffix so the user sees
	// immediate feedback without losing context of the source list.
	if m.state == stateProjectList && m.sessionsLoading {
		projName := filepath.Base(m.loadingProject)
		return breadcrumb("Projects", projName, "Sessions") + "  " +
			hintStyle.Render(m.spinner.View()+" loading…")
	}
	switch m.state {
	case stateProjectList:
		return breadcrumb("Projects")
	case stateProjectAdding:
		return breadcrumb("Projects", "Add project")
	case stateProjectOptions:
		return breadcrumb("Projects", filepath.Base(m.projOptionsTarget), "Options")
	case stateSessions:
		return breadcrumb("Projects", m.sessionsTarget, "Sessions")
	case stateSessionDeleting:
		return breadcrumb("Projects", m.sessionsTarget, "Sessions", m.deleteTarget, "Delete")
	case stateGeneralOptions:
		return breadcrumb("Projects", "General options")
	case stateProfileList:
		return breadcrumb("Projects", "Profiles")
	case stateProfileAdding:
		return breadcrumb("Projects", "Profiles", "Add profile")
	case stateProfileRenaming:
		return breadcrumb("Projects", "Profiles", m.profileRenameTarget, "Rename")
	case stateProfileOptions:
		return breadcrumb("Projects", "Profiles", m.profileOptionsTarget, "Options")
	case stateProfileDeleting:
		return breadcrumb("Projects", "Profiles", m.deleteTarget, "Delete")
	}
	return ""
}

// rightBar builds the app-name + version + optional update indicator.
func (m *Model) rightBar() string {
	out := appNameStyle.Render("llmux") + " " + versionStyle.Render(m.version)
	if m.latestVersion != "" {
		out += "  " + updateStyle.Render(m.latestVersion+" available")
	}
	return out
}

// --- Apply helpers ---

func (m *Model) applyProfileAdd() {
	name := m.profileAddData.Name
	if err := m.cfg.AddProfile(name); err != nil {
		return
	}
	config.Save(m.cfg)

	// Sync settings to the new profile
	m.statusMsg = ""
	if err := config.SyncProfileSettings(m.cfg, name); err != nil {
		m.statusMsg = fmt.Sprintf("settings sync error: %v", err)
	}
}

func (m *Model) applyProfileRename() {
	oldName := m.profileRenameTarget
	newName := strings.TrimSpace(m.profileRenameData.Name)
	if err := m.cfg.RenameProfile(oldName, newName); err != nil {
		m.statusMsg = fmt.Sprintf("rename error: %v", err)
		return
	}
	if err := config.RenameSessionDir(oldName, newName); err != nil {
		_ = m.cfg.RenameProfile(newName, oldName) // roll back in-memory mutation
		m.statusMsg = fmt.Sprintf("rename session dir error: %v", err)
		return
	}
	if err := config.Save(m.cfg); err != nil {
		// Roll back: rename session dir back, revert in-memory config
		if rbErr := config.RenameSessionDir(newName, oldName); rbErr != nil {
			m.statusMsg = fmt.Sprintf("save error and rollback failed — session dir is now %q: %v / %v", newName, err, rbErr)
			return
		}
		_ = m.cfg.RenameProfile(newName, oldName)
		m.statusMsg = fmt.Sprintf("save error: %v", err)
		return
	}
	m.statusMsg = ""
	if err := config.SyncProfileSettings(m.cfg, newName); err != nil {
		m.statusMsg = fmt.Sprintf("settings sync error: %v", err)
	}
}

func (m *Model) applyProfileOptions(name string) {
	// Update worktree setting
	for i := range m.cfg.Profiles {
		if m.cfg.Profiles[i].Name == name {
			m.cfg.Profiles[i].Worktree = m.optionsData.Worktree == "enabled"
			break
		}
	}
	config.Save(m.cfg)

	// Update session settings (attribution)
	if err := config.SetAttribution(name, m.optionsData.DisableAttribution == "enabled"); err != nil {
		m.statusMsg = fmt.Sprintf("attribution error: %v", err)
	}
}

func (m *Model) applyProjAdd() {
	path := expandPath(m.projAddData.FolderPath)
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	if err := m.cfg.AddProject(abs, m.projAddData.Profile); err != nil {
		m.statusMsg = fmt.Sprintf("add project error: %v", err)
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

func (m *Model) refreshProjectList() {
	m.list.SetItems(projectItems(m.cfg))
}

func (m *Model) refreshProfileList() {
	m.profileList.SetItems(profileItems(m.cfg))
}
