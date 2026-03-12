package config

import (
	"encoding/json"
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

func SessionDir(name string) string {
	return filepath.Join(ConfigDir(), "sessions", name)
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
	return os.WriteFile(ConfigFile(), data, 0644)
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
