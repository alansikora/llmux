package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ConfigDir() string {
	if dir := os.Getenv("LLMUX_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "llmux")
}

func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func SessionsDir() string {
	return filepath.Join(ConfigDir(), "sessions")
}

func SessionDir(name string) string {
	return filepath.Join(SessionsDir(), name)
}

func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Migrate legacy format: workspaces with "path" field → projects
	type legacyWorkspace struct {
		Name     string `json:"name"`
		Path     string `json:"path,omitempty"`
		Worktree bool   `json:"worktree,omitempty"`
	}
	type legacyConfig struct {
		Workspaces []legacyWorkspace `json:"workspaces"`
	}

	var legacy legacyConfig
	if err := json.Unmarshal(data, &legacy); err == nil {
		needsMigration := false
		for _, lws := range legacy.Workspaces {
			if lws.Path != "" {
				needsMigration = true
				break
			}
		}

		if needsMigration {
			cfg.Workspaces = nil
			for _, lws := range legacy.Workspaces {
				cfg.Workspaces = append(cfg.Workspaces, Workspace{
					Name:     lws.Name,
					Worktree: lws.Worktree,
				})
				if lws.Path != "" {
					cfg.Projects = append(cfg.Projects, Project{
						Path:      lws.Path,
						Workspace: lws.Name,
					})
				}
			}
			if err := Save(&cfg); err != nil {
				fmt.Fprintf(os.Stderr, "llmux: warning: failed to save migrated config (will retry on next launch): %v\n", err)
			}
		}
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to temp file in same directory, then rename.
	target := ConfigFile()
	tmp, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, target)
}

// RenameSessionDir renames the session directory from oldName to newName.
// It is a no-op if the old directory does not exist.
func RenameSessionDir(oldName, newName string) error {
	oldDir := SessionDir(oldName)
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return nil
	}
	newDir := SessionDir(newName)
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("session directory %q already exists", newDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking session directory %q: %w", newDir, err)
	}
	return os.Rename(oldDir, newDir)
}

// RemoveSessionDir removes the session directory for a workspace.
// It is a no-op if the directory does not exist.
func RemoveSessionDir(name string) error {
	dir := SessionDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(dir)
}

// WriteSessionSettings writes a settings.json into the session directory.
func WriteSessionSettings(name string, settings map[string]any) error {
	dir := SessionDir(name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "settings.json"), data, 0644)
}

// ReadSessionSettings reads the settings.json from the session directory.
func ReadSessionSettings(name string) map[string]any {
	data, err := os.ReadFile(filepath.Join(SessionDir(name), "settings.json"))
	if err != nil {
		return nil
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}
	return settings
}

// newStatusLineConfig returns a fresh copy of the ccstatusline configuration.
func newStatusLineConfig() map[string]any {
	return map[string]any{
		"type":    "command",
		"command": "bunx -y ccstatusline@latest",
		"padding": 0,
	}
}

// SyncWorkspaceSettings writes all llmux-managed keys to a workspace's
// settings.json. It is idempotent and preserves unmanaged keys.
// The write is skipped if the resulting JSON is identical to what is on disk.
func SyncWorkspaceSettings(cfg *Config, wsName string) error {
	existing, _ := os.ReadFile(filepath.Join(SessionDir(wsName), "settings.json"))

	var settings map[string]any
	if len(existing) > 0 {
		json.Unmarshal(existing, &settings) //nolint:errcheck
	}
	if settings == nil {
		settings = map[string]any{}
	}

	// statusLine
	if cfg.StatusLine {
		settings["statusLine"] = newStatusLineConfig()
	} else {
		delete(settings, "statusLine")
	}

	// permissions.defaultMode (auto mode)
	// If permissions exists but is not an object, replace it so we can
	// manage the defaultMode key reliably.
	if raw, exists := settings["permissions"]; exists {
		if _, ok := raw.(map[string]any); !ok {
			fmt.Fprintf(os.Stderr, "llmux: warning: workspace %s has non-object permissions in settings.json, resetting\n", wsName)
			delete(settings, "permissions")
		}
	}
	if cfg.AutoMode {
		perms, _ := settings["permissions"].(map[string]any)
		if perms == nil {
			perms = map[string]any{}
		}
		perms["defaultMode"] = "auto"
		settings["permissions"] = perms
	} else {
		if perms, ok := settings["permissions"].(map[string]any); ok {
			delete(perms, "defaultMode")
			if len(perms) == 0 {
				delete(settings, "permissions")
			}
		}
	}

	// Skip write if nothing changed to avoid unnecessary I/O and write races.
	// Normalize existing bytes through marshal to make comparison format-independent.
	updated, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	var existingParsed map[string]any
	if json.Unmarshal(existing, &existingParsed) == nil {
		existingNorm, _ := json.MarshalIndent(existingParsed, "", "  ")
		if string(updated) == string(existingNorm) {
			return nil
		}
	}
	return WriteSessionSettings(wsName, settings)
}

// SyncAllWorkspaceSettings syncs settings for all workspaces.
func SyncAllWorkspaceSettings(cfg *Config) error {
	var errs []error
	for _, ws := range cfg.Workspaces {
		if err := SyncWorkspaceSettings(cfg, ws.Name); err != nil {
			errs = append(errs, fmt.Errorf("workspace %s: %w", ws.Name, err))
		}
	}
	return errors.Join(errs...)
}

// IsAttributionDisabled reports whether the attribution-suppression key
// is present in a workspace's session settings.
func IsAttributionDisabled(name string) bool {
	settings := ReadSessionSettings(name)
	if settings == nil {
		return false
	}
	_, ok := settings["attribution"]
	return ok
}

// SetAttribution enables or disables the attribution setting for a workspace.
func SetAttribution(name string, disabled bool) error {
	settings := ReadSessionSettings(name)
	if settings == nil {
		settings = map[string]any{}
	}
	if disabled {
		settings["attribution"] = map[string]string{
			"commit": "",
			"pr":     "",
		}
	} else {
		delete(settings, "attribution")
	}
	return WriteSessionSettings(name, settings)
}

// AuthInfo holds display information about a workspace's authentication.
type AuthInfo struct {
	Authenticated bool
	Email         string
	Organization  string
}

// GetAuthInfo reads the .claude.json in the session directory to determine
// authentication status and account details.
func GetAuthInfo(name string) AuthInfo {
	data, err := os.ReadFile(filepath.Join(SessionDir(name), ".claude.json"))
	if err != nil {
		return AuthInfo{}
	}
	var doc struct {
		OAuthAccount *struct {
			EmailAddress     string `json:"emailAddress"`
			OrganizationName string `json:"organizationName"`
		} `json:"oauthAccount"`
	}
	if err := json.Unmarshal(data, &doc); err != nil || doc.OAuthAccount == nil {
		return AuthInfo{}
	}
	return AuthInfo{
		Authenticated: true,
		Email:         doc.OAuthAccount.EmailAddress,
		Organization:  doc.OAuthAccount.OrganizationName,
	}
}

// IsAuthenticated reports whether a workspace has valid auth credentials.
func IsAuthenticated(name string) bool {
	return GetAuthInfo(name).Authenticated
}
