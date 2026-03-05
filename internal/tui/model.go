package tui

import (
	"path/filepath"

	"github.com/allskar/llmux/internal/config"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateList state = iota
	stateAdding
	stateDeleting
)

type Model struct {
	cfg   *config.Config
	state state

	list   list.Model
	width  int
	height int

	// Add form
	addForm     *huh.Form
	addData     addFormData

	// Delete form
	deleteForm   *huh.Form
	deleteData   deleteFormData
	deleteTarget string
}

func NewModel(cfg *config.Config) *Model {
	return &Model{
		cfg:   cfg,
		state: stateList,
	}
}

func (m *Model) Init() tea.Cmd {
	m.list = buildList(m.cfg, 80, 20)
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
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
			if m.addData.Confirm {
				m.applyAdd()
			}
			m.state = stateList
			m.refreshList()
			return m, nil
		}
		if m.addForm.State == huh.StateAborted {
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
	}

	return m, nil
}

func (m *Model) View() string {
	var content string

	switch m.state {
	case stateList:
		content = m.list.View()
	case stateAdding:
		content = titleStyle.Render("Add Workspace") + "\n\n" + m.addForm.View()
	case stateDeleting:
		content = m.deleteForm.View()
	}

	return appStyle.Render(lipgloss.Place(
		m.width, m.height,
		lipgloss.Left, lipgloss.Top,
		content,
	))
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
	config.Save(m.cfg)

	settings := map[string]any{}
	if m.addData.DisableAttribution {
		settings["attribution"] = map[string]string{
			"commit": "",
			"pr":     "",
		}
	}
	if len(settings) > 0 {
		config.WriteSessionSettings(name, settings)
	}
}

func (m *Model) refreshList() {
	items := make([]list.Item, len(m.cfg.Workspaces))
	for i, ws := range m.cfg.Workspaces {
		items[i] = workspaceItem{
			name: ws.Name,
			path: ws.Path,
			auth: config.IsAuthenticated(ws.Name),
		}
	}
	m.list.SetItems(items)
}
