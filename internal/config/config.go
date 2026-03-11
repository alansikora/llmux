package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Workspace struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	APIKey string `json:"api_key,omitempty"`
}

type Config struct {
	Workspaces       []Workspace `json:"workspaces"`
	DefaultWorkspace string      `json:"default_workspace,omitempty"`
}

type ResolveResult struct {
	SessionDir string
	APIKey     string
}

func (c *Config) Add(name, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)

	for _, ws := range c.Workspaces {
		if ws.Name == name {
			return fmt.Errorf("workspace %q already exists", name)
		}
		if ws.Path == abs {
			return fmt.Errorf("path %q is already assigned to workspace %q", abs, ws.Name)
		}
	}

	c.Workspaces = append(c.Workspaces, Workspace{Name: name, Path: abs})
	return nil
}

func (c *Config) Remove(name string) error {
	for i, ws := range c.Workspaces {
		if ws.Name == name {
			c.Workspaces = append(c.Workspaces[:i], c.Workspaces[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("workspace %q not found", name)
}

// Resolve returns the session directory and optional API key for the workspace
// that best matches the given path using longest-prefix match with
// path-separator boundary.
func (c *Config) Resolve(dir string) (ResolveResult, error) {
	dir = filepath.Clean(dir)

	var best *Workspace
	bestLen := 0

	for i := range c.Workspaces {
		ws := &c.Workspaces[i]
		if dir == ws.Path || strings.HasPrefix(dir, ws.Path+"/") {
			if len(ws.Path) > bestLen {
				best = ws
				bestLen = len(ws.Path)
			}
		}
	}

	if best == nil {
		if c.DefaultWorkspace != "" {
			for i := range c.Workspaces {
				if c.Workspaces[i].Name == c.DefaultWorkspace {
					return ResolveResult{
						SessionDir: SessionDir(c.Workspaces[i].Name),
						APIKey:     c.Workspaces[i].APIKey,
					}, nil
				}
			}
		}
		return ResolveResult{}, fmt.Errorf("no workspace configured for %s", dir)
	}

	return ResolveResult{
		SessionDir: SessionDir(best.Name),
		APIKey:     best.APIKey,
	}, nil
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
