package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/allskar/llmux/internal/config"
)

type workspaceItem struct {
	name string
	path string
	auth bool
}

func (w workspaceItem) Title() string {
	indicator := unauthStyle.Render("○")
	if w.auth {
		indicator = authStyle.Render("●")
	}
	return fmt.Sprintf("%s %s", indicator, w.name)
}

func (w workspaceItem) Description() string { return w.path }
func (w workspaceItem) FilterValue() string { return w.name }

func buildList(cfg *config.Config, width, height int) list.Model {
	items := make([]list.Item, len(cfg.Workspaces))
	for i, ws := range cfg.Workspaces {
		items[i] = workspaceItem{
			name: ws.Name,
			path: ws.Path,
			auth: config.IsAuthenticated(ws.Name),
		}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, width, height)
	l.Title = "Workspaces"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "delete")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	return l
}

func updateList(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a":
			m.state = stateAdding
			m.addData = addFormData{}
			m.addForm = newAddForm(&m.addData)
			return m, m.addForm.Init()
		case "d", "x":
			if item, ok := m.list.SelectedItem().(workspaceItem); ok {
				m.state = stateDeleting
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
