package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/allskar/llmux/internal/config"
)

type workspaceItem struct {
	name      string
	path      string
	authInfo  config.AuthInfo
	isDefault bool
	loading   string
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

func (w workspaceItem) Description() string { return w.path }
func (w workspaceItem) FilterValue() string { return w.name }

const logo = ` _ _
| | |_ __ ___  _   ___  __
| | | '_ ` + "`" + ` _ \| | | \ \/ /
| | | | | | | | |_| |>  <
|_|_|_| |_| |_|\__,_/_/\_\`

func buildList(cfg *config.Config, version string, width, height int) list.Model {
	items := make([]list.Item, len(cfg.Workspaces))
	for i, ws := range cfg.Workspaces {
		items[i] = workspaceItem{
			name:      ws.Name,
			path:      ws.Path,
			authInfo:  config.GetAuthInfo(ws.Name),
			isDefault: ws.Name == cfg.DefaultWorkspace,
		}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, width, height)
	l.Title = "Workspaces"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "options")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "set default")),
			key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "worktrees")),
			key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "general options")),
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
		case "enter":
			if item, ok := m.list.SelectedItem().(workspaceItem); ok {
				// Pre-populate from current settings
				settings := config.ReadSessionSettings(item.name)
				disableAttribution := false
				if settings != nil {
					if _, ok := settings["attribution"]; ok {
						disableAttribution = true
					}
				}
				// Find worktree setting from config
				alwaysWorktree := false
				for _, ws := range m.cfg.Workspaces {
					if ws.Name == item.name {
						alwaysWorktree = ws.Worktree
						break
					}
				}
				m.optionsTarget = item.name
				m.optionsData = optionsFormData{
					DisableAttribution: disableAttribution,
					AlwaysWorktree:     alwaysWorktree,
				}
				m.optionsForm = newOptionsForm(&m.optionsData, m.optionsData)
				m.state = stateOptions
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
				m.refreshList()
				return m, nil
			}
		case "w":
			if item, ok := m.list.SelectedItem().(workspaceItem); ok {
				m.sessionsLoading = true
				m.loadingWorkspace = item.name
				m.updateLoadingItem()
				return m, tea.Batch(loadSessionsCmd(item.path, item.name), m.spinner.Tick)
			}
		case "o":
			m.generalOptionsData = generalOptionsFormData{
				ShortAlias: m.cfg.ShortAlias,
			}
			m.generalOptionsForm = newGeneralOptionsForm(&m.generalOptionsData, m.generalOptionsData)
			m.state = stateGeneralOptions
			return m, m.generalOptionsForm.Init()
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
