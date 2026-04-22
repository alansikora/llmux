package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var ErrUnmapped = errors.New("no project configured for this directory")

type Profile struct {
	Name     string `json:"name"`
	Worktree bool   `json:"worktree,omitempty"`
}

type ProjectOverrides struct {
	Worktree *bool `json:"worktree,omitempty"` // nil = inherit from profile
}

type Project struct {
	Path      string           `json:"path"`
	Profile   string           `json:"profile"`
	Overrides ProjectOverrides `json:"overrides,omitempty"`
}

// ResolvedWorktree returns the effective worktree setting for this project,
// using the profile default if no override is set.
func (p *Project) ResolvedWorktree(pf Profile) bool {
	if p.Overrides.Worktree != nil {
		return *p.Overrides.Worktree
	}
	return pf.Worktree
}

type Config struct {
	Profiles           []Profile `json:"profiles"`
	Projects           []Project `json:"projects,omitempty"`
	DefaultProfile     string    `json:"default_profile,omitempty"`
	ShortAlias         bool      `json:"short_alias,omitempty"`
	ApplyMarker        bool      `json:"apply_marker,omitempty"`
	AutoMode           bool      `json:"auto_mode,omitempty"`
	StatusLine         bool      `json:"status_line,omitempty"`
	AutoDefaultProfile bool      `json:"auto_default_profile,omitempty"`
}

type ResolveResult struct {
	SessionDir  string
	Worktree    bool
	ProfileName string
	ProjectPath string // empty if resolved via default profile fallback
}

func (c *Config) AddProfile(name string) error {
	for _, pf := range c.Profiles {
		if pf.Name == name {
			return fmt.Errorf("profile %q already exists", name)
		}
	}
	c.Profiles = append(c.Profiles, Profile{Name: name})
	return nil
}

func (c *Config) RenameProfile(oldName, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if oldName == newName {
		return nil
	}
	for _, pf := range c.Profiles {
		if pf.Name == newName {
			return fmt.Errorf("profile %q already exists", newName)
		}
	}
	found := false
	for i := range c.Profiles {
		if c.Profiles[i].Name == oldName {
			c.Profiles[i].Name = newName
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("profile %q not found", oldName)
	}
	for i := range c.Projects {
		if c.Projects[i].Profile == oldName {
			c.Projects[i].Profile = newName
		}
	}
	if c.DefaultProfile == oldName {
		c.DefaultProfile = newName
	}
	return nil
}

func (c *Config) RemoveProfile(name string) error {
	found := false
	for i, pf := range c.Profiles {
		if pf.Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("profile %q not found", name)
	}
	// Remove associated projects
	var remaining []Project
	for _, p := range c.Projects {
		if p.Profile != name {
			remaining = append(remaining, p)
		}
	}
	c.Projects = remaining
	if c.DefaultProfile == name {
		c.DefaultProfile = ""
	}
	return nil
}

func (c *Config) AddProject(path, profileName string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)

	// Verify profile exists
	if _, err := c.FindProfile(profileName); err != nil {
		return err
	}

	// Check for duplicate path
	for _, p := range c.Projects {
		if p.Path == abs {
			return fmt.Errorf("path %q is already registered (profile: %s)", abs, p.Profile)
		}
	}

	c.Projects = append(c.Projects, Project{Path: abs, Profile: profileName})
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

func (c *Config) SetProjectProfile(path, profileName string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)

	// Verify profile exists
	if _, err := c.FindProfile(profileName); err != nil {
		return err
	}

	for i, p := range c.Projects {
		if p.Path == abs {
			c.Projects[i].Profile = profileName
			return nil
		}
	}
	return fmt.Errorf("no project at %q", abs)
}

func (c *Config) ProjectsForProfile(name string) []Project {
	var result []Project
	for _, p := range c.Projects {
		if p.Profile == name {
			result = append(result, p)
		}
	}
	return result
}

func (c *Config) FindProfile(name string) (*Profile, error) {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("profile %q not found", name)
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

// FindProfileForDir finds the profile for a given directory by looking up
// the project first, then its profile. When AutoDefaultProfile is enabled,
// falls back to the default profile silently; otherwise returns ErrUnmapped
// so the shell wrapper can prompt the user to register the directory.
func (c *Config) FindProfileForDir(dir string) (*Profile, *Project, error) {
	proj, err := c.FindProject(dir)
	if err == nil {
		pf, pfErr := c.FindProfile(proj.Profile)
		if pfErr != nil {
			return nil, nil, pfErr
		}
		return pf, proj, nil
	}

	// Silent fallback to default is opt-in. Otherwise, surface ErrUnmapped
	// so the shell wrapper triggers `llmux register` (which pre-selects the
	// default profile for one-keystroke confirmation).
	if c.AutoDefaultProfile {
		if c.DefaultProfile == "" {
			return nil, nil, fmt.Errorf("auto_default_profile is enabled but no default profile is set")
		}
		pf, pfErr := c.FindProfile(c.DefaultProfile)
		if pfErr != nil {
			return nil, nil, fmt.Errorf("auto_default_profile points at %q which does not exist", c.DefaultProfile)
		}
		return pf, nil, nil
	}

	return nil, nil, ErrUnmapped
}

// Resolve returns the session directory and config for the profile
// that best matches the given path through project lookup.
func (c *Config) Resolve(dir string) (ResolveResult, error) {
	pf, proj, err := c.FindProfileForDir(dir)
	if err != nil {
		return ResolveResult{}, err
	}

	result := ResolveResult{
		SessionDir:  SessionDir(pf.Name),
		Worktree:    pf.Worktree,
		ProfileName: pf.Name,
	}
	if proj != nil {
		result.Worktree = proj.ResolvedWorktree(*pf)
		result.ProjectPath = proj.Path
	}
	return result, nil
}

// ResolveProfile returns a ResolveResult for the named profile, bypassing
// directory-based project lookup. Used when the user explicitly pins a profile
// (e.g. via LLMUX_PROFILE) for a single invocation. Worktree mode falls back to
// the profile default since no project context applies.
func (c *Config) ResolveProfile(name string) (ResolveResult, error) {
	pf, err := c.FindProfile(name)
	if err != nil {
		return ResolveResult{}, err
	}
	return ResolveResult{
		SessionDir:  SessionDir(pf.Name),
		Worktree:    pf.Worktree,
		ProfileName: pf.Name,
	}, nil
}

func (c *Config) SetDefaultProfile(name string) error {
	if name == "" {
		c.DefaultProfile = ""
		return nil
	}
	for _, pf := range c.Profiles {
		if pf.Name == name {
			c.DefaultProfile = name
			return nil
		}
	}
	return fmt.Errorf("profile %q not found", name)
}
