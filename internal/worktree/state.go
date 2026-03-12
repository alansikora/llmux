package worktree

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type ApplyState struct {
	Session      string    `json:"session"`
	StashCreated bool      `json:"stash_created"`
	AppliedAt    time.Time `json:"applied_at"`
}

func stateFile(workspacePath string) string {
	return filepath.Join(workspacePath, ".claude", "worktrees", ".llmux-apply-state.json")
}

func SaveState(workspacePath string, state ApplyState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile(workspacePath), data, 0644)
}

func LoadState(workspacePath string) (*ApplyState, error) {
	data, err := os.ReadFile(stateFile(workspacePath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var state ApplyState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func RemoveState(workspacePath string) error {
	err := os.Remove(stateFile(workspacePath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
