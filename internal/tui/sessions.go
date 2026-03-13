package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/allskar/llmux/internal/worktree"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type sessionItem struct {
	name          string
	branch        string
	changedFiles  int
	lastActivity  time.Time
	applied       bool
	stale         bool
	workspacePath string
	path          string
}

type clipboardResultMsg struct {
	err  error
	path string
}

func (s sessionItem) Title() string {
	indicator := "  "
	if s.applied {
		indicator = appliedStyle.Render("▶ ")
	}
	return fmt.Sprintf("%s%s", indicator, s.name)
}

func (s sessionItem) Description() string {
	desc := fmt.Sprintf("%s · %d files changed", s.branch, s.changedFiles)
	if !s.lastActivity.IsZero() {
		age := relativeTime(s.lastActivity)
		if s.stale {
			desc += staleStyle.Render(fmt.Sprintf(" · %s (stale)", age))
		} else {
			desc += fmt.Sprintf(" · %s", age)
		}
	}
	desc += "\n" + dimStyle.Render(s.path)
	return desc
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w ago"
		}
		return fmt.Sprintf("%dw ago", weeks)
	}
}

func (s sessionItem) FilterValue() string { return s.name }

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Messages for async operations
type sessionsLoadedMsg struct {
	sessions []worktree.Session
	applied  string
	wsPath   string
	target   string
}

type applyResultMsg struct {
	err     error
	session string
}

type unapplyResultMsg struct {
	err error
}

type deleteResultMsg struct {
	err     error
	session string
}

func loadSessionsCmd(rawPath, target string) tea.Cmd {
	return func() tea.Msg {
		wsPath := worktree.ResolveSessionsPath(rawPath)
		sessions, _ := worktree.ListSessions(wsPath)
		_, applied, _ := worktree.FindAppliedWorkspace(sessions)
		return sessionsLoadedMsg{sessions: sessions, applied: applied, wsPath: wsPath, target: target}
	}
}

func deleteSessionCmd(wsPath, sessionName string, force bool) tea.Cmd {
	return func() tea.Msg {
		err := worktree.Delete(wsPath, sessionName, force)
		return deleteResultMsg{err: err, session: sessionName}
	}
}

func applySessionCmd(wsPath, sessionName string, applyMarker bool) tea.Cmd {
	return func() tea.Msg {
		err := worktree.Apply(wsPath, sessionName, applyMarker)
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
			name:          s.Name,
			branch:        s.Branch,
			changedFiles:  s.ChangedFiles,
			lastActivity:  s.LastActivity,
			applied:       s.Name == applied,
			stale:         s.IsStale(),
			workspacePath: s.WorkspacePath,
			path:          s.Path,
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(3)
	l := list.New(items, delegate, width, height)
	l.Title = "Worktree Sessions"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a", "enter"), key.WithHelp("a/enter", "apply")),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unapply")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy path")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	return l
}

func sessionsFromList(l list.Model) []worktree.Session {
	items := l.Items()
	sessions := make([]worktree.Session, 0, len(items))
	for _, item := range items {
		if s, ok := item.(sessionItem); ok {
			sessions = append(sessions, worktree.Session{
				Name:          s.name,
				WorkspacePath: s.workspacePath,
			})
		}
	}
	return sessions
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
				return m, applySessionCmd(item.workspacePath, item.name, m.cfg.ApplyMarker)
			}
		case "u":
			wsPath, applied, ok := worktree.FindAppliedWorkspace(sessionsFromList(m.sessionsList))
			if !ok {
				m.sessionsStatus = "nothing to unapply"
				return m, nil
			}
			m.sessionsStatus = fmt.Sprintf("unapplying %s...", applied)
			return m, unapplySessionCmd(wsPath)
		case "c":
			if item, ok := m.sessionsList.SelectedItem().(sessionItem); ok {
				path := item.path
				return m, func() tea.Msg {
					err := copyToClipboard(path)
					return clipboardResultMsg{err: err, path: path}
				}
			}
		case "d":
			if item, ok := m.sessionsList.SelectedItem().(sessionItem); ok {
				m.sessionsStatus = fmt.Sprintf("deleting %s...", item.name)
				return m, deleteSessionCmd(item.workspacePath, item.name, false)
			}
		}
	}

	var cmd tea.Cmd
	m.sessionsList, cmd = m.sessionsList.Update(msg)
	return m, cmd
}
