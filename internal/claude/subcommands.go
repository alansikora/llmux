package claude

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/allskar/llmux/internal/config"
)

const subcmdCacheFile = "claude-subcommands.json"

type subcmdCache struct {
	Subcommands []string  `json:"subcommands"`
	CachedAt    time.Time `json:"cached_at"`
}

func subcmdCachePath() string {
	return filepath.Join(config.ConfigDir(), subcmdCacheFile)
}

func readSubcmdCacheWithTTL(ttl time.Duration) map[string]bool {
	data, err := os.ReadFile(subcmdCachePath())
	if err != nil {
		return nil
	}
	var c subcmdCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	if ttl > 0 && time.Since(c.CachedAt) > ttl {
		return nil
	}
	if len(c.Subcommands) == 0 {
		return nil
	}
	m := make(map[string]bool, len(c.Subcommands))
	for _, s := range c.Subcommands {
		m[s] = true
	}
	return m
}

func writeSubcmdCache(subcmds map[string]bool) {
	list := make([]string, 0, len(subcmds))
	for s := range subcmds {
		list = append(list, s)
	}
	data, err := json.Marshal(subcmdCache{Subcommands: list, CachedAt: time.Now()})
	if err != nil {
		return
	}
	p := subcmdCachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0644)
}

// ParseSubcommands parses the Commands section from `claude --help` output.
func ParseSubcommands(output string) map[string]bool {
	subcmds := make(map[string]bool)
	lines := strings.Split(output, "\n")
	inCommands := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "Commands:" {
			inCommands = true
			continue
		}
		if !inCommands {
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		name := strings.Fields(trimmed)[0]
		for _, n := range strings.Split(name, "|") {
			subcmds[n] = true
		}
	}
	return subcmds
}

// GetSubcommands returns the set of claude CLI subcommands, using a
// cached result when available (24h TTL) to avoid spawning a subprocess on
// every invocation.
func GetSubcommands() map[string]bool {
	if cached := readSubcmdCacheWithTTL(24 * time.Hour); cached != nil {
		return cached
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "claude", "--help").CombinedOutput()
	if err != nil {
		return readSubcmdCacheWithTTL(0) // stale fallback
	}

	subcmds := ParseSubcommands(string(out))
	if len(subcmds) > 0 {
		writeSubcmdCache(subcmds)
	}
	return subcmds
}

// IsSubcommand checks whether the first positional argument in the
// claude args matches a real claude subcommand (queried from `claude --help`).
// If we can't determine the subcommands, we assume it's an interactive session.
func IsSubcommand(claudeArgs []string) bool {
	for _, a := range claudeArgs {
		if strings.HasPrefix(a, "-") {
			continue
		}
		subcmds := GetSubcommands()
		if subcmds == nil {
			return false
		}
		return subcmds[a]
	}
	return false
}
