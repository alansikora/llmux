package tui

import (
	"fmt"

	"github.com/allskar/llmux/internal/worktree"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type sessionItem struct {
	name         string
	branch       string
	changedFiles int
	applied      bool
}

func (s sessionItem) Title() string {
	indicator := "  "
	if s.applied {
		indicator = appliedStyle.Render("▶ ")
	}
	return fmt.Sprintf("%s%s", indicator, s.name)
}

func (s sessionItem) Description() string {
	return fmt.Sprintf("%s · %d files changed", s.branch, s.changedFiles)
}

func (s sessionItem) FilterValue() string { return s.name }

// Messages for async operations
type sessionsLoadedMsg struct {
	sessions []worktree.Session
	applied  string
}

type applyResultMsg struct {
	err     error
	session string
}

type unapplyResultMsg struct {
	err error
}

func loadSessionsCmd(wsPath string) tea.Cmd {
	return func() tea.Msg {
		sessions, _ := worktree.ListSessions(wsPath)
		applied, _ := worktree.HasAppliedSession(wsPath)
		return sessionsLoadedMsg{sessions: sessions, applied: applied}
	}
}

func applySessionCmd(wsPath, sessionName string) tea.Cmd {
	return func() tea.Msg {
		err := worktree.Apply(wsPath, sessionName)
		return applyResultMsg{err: err, session: sessionName}
	}
}

func unapplySessionCmd(wsPath string) tea.Cmd {
	return func() tea.Msg {
		err := worktree.Unapply(wsPath)
		return unapplyResultMsg{err: err}
	}
}

func buildSessionsList(sessions []worktree.Session, applied string, width, height int) list.Model {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{
			name:         s.Name,
			branch:       s.Branch,
			changedFiles: s.ChangedFiles,
			applied:      s.Name == applied,
		}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, width, height)
	l.Title = "Worktree Sessions"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a", "enter"), key.WithHelp("a/enter", "apply")),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unapply")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	return l
}

func updateSessions(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.sessionsList.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a", "enter":
			if item, ok := m.sessionsList.SelectedItem().(sessionItem); ok {
				if item.applied {
					m.sessionsStatus = "already applied"
					return m, nil
				}
				m.sessionsStatus = fmt.Sprintf("applying %s...", item.name)
				return m, applySessionCmd(m.sessionsPath, item.name)
			}
		case "u":
			applied, ok := worktree.HasAppliedSession(m.sessionsPath)
			if !ok {
				m.sessionsStatus = "nothing to unapply"
				return m, nil
			}
			m.sessionsStatus = fmt.Sprintf("unapplying %s...", applied)
			return m, unapplySessionCmd(m.sessionsPath)
		}
	}

	var cmd tea.Cmd
	m.sessionsList, cmd = m.sessionsList.Update(msg)
	return m, cmd
}
