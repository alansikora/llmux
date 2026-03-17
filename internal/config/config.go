package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var ErrUnmapped = errors.New("no project configured for this directory")

type Workspace struct {
	Name     string `json:"name"`
	APIKey   string `json:"api_key,omitempty"`
	Worktree bool   `json:"worktree,omitempty"`
}

type ProjectOverrides struct {
	Worktree *bool `json:"worktree,omitempty"` // nil = inherit from workspace
}

type Project struct {
	Path      string           `json:"path"`
	Workspace string           `json:"workspace"`
	Overrides ProjectOverrides `json:"overrides,omitempty"`
}

// ResolvedWorktree returns the effective worktree setting for this project,
// using the workspace default if no override is set.
func (p *Project) ResolvedWorktree(ws Workspace) bool {
	if p.Overrides.Worktree != nil {
		return *p.Overrides.Worktree
	}
	return ws.Worktree
}

type Config struct {
	Workspaces       []Workspace `json:"workspaces"`
	Projects         []Project   `json:"projects,omitempty"`
	DefaultWorkspace string      `json:"default_workspace,omitempty"`
	ShortAlias       bool        `json:"short_alias,omitempty"`
	ApplyMarker      bool        `json:"apply_marker,omitempty"`
	AutoMode         bool        `json:"auto_mode,omitempty"`
}

type ResolveResult struct {
	SessionDir    string
	APIKey        string
	Worktree      bool
	AutoMode      bool
	WorkspaceName string
	ProjectPath   string // empty if resolved via default workspace fallback
}

func (c *Config) AddWorkspace(name string) error {
	for _, ws := range c.Workspaces {
		if ws.Name == name {
			return fmt.Errorf("workspace %q already exists", name)
		}
	}
	c.Workspaces = append(c.Workspaces, Workspace{Name: name})
	return nil
}

func (c *Config) RemoveWorkspace(name string) error {
	found := false
	for i, ws := range c.Workspaces {
		if ws.Name == name {
			c.Workspaces = append(c.Workspaces[:i], c.Workspaces[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("workspace %q not found", name)
	}
	// Remove associated projects
	var remaining []Project
	for _, p := range c.Projects {
		if p.Workspace != name {
			remaining = append(remaining, p)
		}
	}
	c.Projects = remaining
	if c.DefaultWorkspace == name {
		c.DefaultWorkspace = ""
	}
	return nil
}

func (c *Config) AddProject(path, workspaceName string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)

	// Verify workspace exists
	wsFound := false
	for _, ws := range c.Workspaces {
		if ws.Name == workspaceName {
			wsFound = true
			break
		}
	}
	if !wsFound {
		return fmt.Errorf("workspace %q not found", workspaceName)
	}

	// Check for duplicate path
	for _, p := range c.Projects {
		if p.Path == abs {
			return fmt.Errorf("path %q is already registered (workspace: %s)", abs, p.Workspace)
		}
	}

	c.Projects = append(c.Projects, Project{Path: abs, Workspace: workspaceName})
	return nil
}

func (c *Config) RemoveProject(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)

	for i, p := range c.Projects {
		if p.Path == abs {
			c.Projects = append(c.Projects[:i], c.Projects[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("no project at %q", abs)
}

func (c *Config) SetProjectWorkspace(path, workspaceName string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)

	// Verify workspace exists
	wsFound := false
	for _, ws := range c.Workspaces {
		if ws.Name == workspaceName {
			wsFound = true
			break
		}
	}
	if !wsFound {
		return fmt.Errorf("workspace %q not found", workspaceName)
	}

	for i, p := range c.Projects {
		if p.Path == abs {
			c.Projects[i].Workspace = workspaceName
			return nil
		}
	}
	return fmt.Errorf("no project at %q", abs)
}

func (c *Config) ProjectsForWorkspace(name string) []Project {
	var result []Project
	for _, p := range c.Projects {
		if p.Workspace == name {
			result = append(result, p)
		}
	}
	return result
}

func (c *Config) FindWorkspace(name string) (*Workspace, error) {
	for i := range c.Workspaces {
		if c.Workspaces[i].Name == name {
			return &c.Workspaces[i], nil
		}
	}
	return nil, fmt.Errorf("workspace %q not found", name)
}

// FindProject finds the project that best matches the given directory
// using longest-prefix match with path-separator boundary.
func (c *Config) FindProject(dir string) (*Project, error) {
	dir = filepath.Clean(dir)

	var best *Project
	bestLen := 0
	for i := range c.Projects {
		p := &c.Projects[i]
		if dir == p.Path || strings.HasPrefix(dir, p.Path+"/") {
			if len(p.Path) > bestLen {
				best = p
				bestLen = len(p.Path)
			}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no project found for %q", dir)
	}
	return best, nil
}

// FindWorkspaceForDir finds the workspace for a given directory by looking up
// the project first, then its workspace. Falls back to default workspace.
func (c *Config) FindWorkspaceForDir(dir string) (*Workspace, *Project, error) {
	proj, err := c.FindProject(dir)
	if err == nil {
		ws, wsErr := c.FindWorkspace(proj.Workspace)
		if wsErr != nil {
			return nil, nil, wsErr
		}
		return ws, proj, nil
	}

	// Fall back to default workspace
	if c.DefaultWorkspace != "" {
		ws, wsErr := c.FindWorkspace(c.DefaultWorkspace)
		if wsErr == nil {
			return ws, nil, nil
		}
	}

	return nil, nil, ErrUnmapped
}

// Resolve returns the session directory and config for the workspace
// that best matches the given path through project lookup.
func (c *Config) Resolve(dir string) (ResolveResult, error) {
	dir = filepath.Clean(dir)

	// Find best matching project
	var bestProject *Project
	bestLen := 0

	for i := range c.Projects {
		p := &c.Projects[i]
		if dir == p.Path || strings.HasPrefix(dir, p.Path+"/") {
			if len(p.Path) > bestLen {
				bestProject = p
				bestLen = len(p.Path)
			}
		}
	}

	if bestProject != nil {
		for i := range c.Workspaces {
			if c.Workspaces[i].Name == bestProject.Workspace {
				ws := &c.Workspaces[i]
				return ResolveResult{
					SessionDir:    SessionDir(ws.Name),
					APIKey:        ws.APIKey,
					Worktree:      bestProject.ResolvedWorktree(*ws),
					AutoMode:      c.AutoMode,
					WorkspaceName: ws.Name,
					ProjectPath:   bestProject.Path,
				}, nil
			}
		}
		return ResolveResult{}, fmt.Errorf("workspace %q not found for project %s", bestProject.Workspace, bestProject.Path)
	}

	// No project match — try default workspace
	if c.DefaultWorkspace != "" {
		for i := range c.Workspaces {
			if c.Workspaces[i].Name == c.DefaultWorkspace {
				ws := &c.Workspaces[i]
				return ResolveResult{
					SessionDir:    SessionDir(ws.Name),
					APIKey:        ws.APIKey,
					Worktree:      ws.Worktree,
					AutoMode:      c.AutoMode,
					WorkspaceName: ws.Name,
				}, nil
			}
		}
	}

	return ResolveResult{}, ErrUnmapped
}

func (c *Config) SetDefault(name string) error {
	if name == "" {
		c.DefaultWorkspace = ""
		return nil
	}
	for _, ws := range c.Workspaces {
		if ws.Name == name {
			c.DefaultWorkspace = name
			return nil
		}
	}
	return fmt.Errorf("workspace %q not found", name)
}
