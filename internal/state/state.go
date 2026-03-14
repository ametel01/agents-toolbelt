// Package state stores persistent tool ownership and verification metadata.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// OwnershipExternal identifies a tool that exists on PATH but is not managed by atb.
	OwnershipExternal = "external"
	// OwnershipManaged identifies a tool installed and owned by atb.
	OwnershipManaged = "managed"
)

var (
	// ErrCorruptState indicates the state file could not be decoded and a fresh state was returned.
	ErrCorruptState  = errors.New("state file is corrupt")
	errMissingToolID = errors.New("tool id is required")
)

// State represents the persisted atb state file contents.
type State struct {
	Version      int                  `json:"version"`
	Tools        map[string]ToolState `json:"tools"`
	SkillTargets []string             `json:"skill_targets,omitempty"`
	LastRunAt    time.Time            `json:"last_run_at"`
}

// ToolState records ownership and verification data for one tool.
type ToolState struct {
	ToolID               string            `json:"tool_id"`
	Bin                  string            `json:"bin"`
	Ownership            string            `json:"ownership"`
	InstallManager       string            `json:"install_manager,omitempty"`
	InstallPackage       string            `json:"install_package,omitempty"`
	InstallCommand       []string          `json:"install_command,omitempty"`
	UpdateCommand        []string          `json:"update_command,omitempty"`
	UninstallCommand     []string          `json:"uninstall_command,omitempty"`
	InstalledAt          time.Time         `json:"installed_at,omitempty"`
	LastUpdateAttemptAt  time.Time         `json:"last_update_attempt_at,omitempty"`
	LastVerifyAt         time.Time         `json:"last_verify_at,omitempty"`
	LastVerifyOK         bool              `json:"last_verify_ok"`
	LastVerifyError      string            `json:"last_verify_error,omitempty"`
	Version              string            `json:"version,omitempty"`
	ShellHookStatus      string            `json:"shell_hook_status,omitempty"`
	ShellHookSuggestedAt time.Time         `json:"shell_hook_suggested_at,omitempty"`
	ShellHookAppliedAt   time.Time         `json:"shell_hook_applied_at,omitempty"`
	BinaryPath           string            `json:"binary_path,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

// DefaultPath returns the canonical atb state file path.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	return filepath.Join(configDir, "atb", "state.json"), nil
}

// Load reads the default atb state file.
func Load() (State, error) {
	path, err := DefaultPath()
	if err != nil {
		return State{}, err
	}

	return loadFromPath(path)
}

// Save persists the state to the default atb state file path.
func Save(st State) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}

	return saveToPath(path, st)
}

// Tool returns the persisted state for a tool ID, if present.
func (s State) Tool(id string) (*ToolState, bool) {
	if len(s.Tools) == 0 {
		return nil, false
	}

	toolState, ok := s.Tools[id]
	if !ok {
		return nil, false
	}

	receipt := toolState

	return &receipt, true
}

// AddReceipt stores a managed tool receipt.
func (s *State) AddReceipt(receipt ToolState) error {
	if receipt.ToolID == "" {
		return errMissingToolID
	}

	receipt.Ownership = OwnershipManaged

	return s.SetTool(receipt)
}

// MarkExternal records that a tool exists but is not managed by atb.
func (s *State) MarkExternal(toolID, bin, binaryPath string) error {
	if toolID == "" {
		return errMissingToolID
	}

	return s.SetTool(ToolState{
		ToolID:     toolID,
		Bin:        bin,
		Ownership:  OwnershipExternal,
		BinaryPath: binaryPath,
	})
}

// SetTool stores tool state without changing ownership semantics.
func (s *State) SetTool(toolState ToolState) error {
	if toolState.ToolID == "" {
		return errMissingToolID
	}

	s.ensureInitialized()
	s.Tools[toolState.ToolID] = toolState

	return nil
}

// IsATBManaged reports whether a tool has a managed receipt.
func (s State) IsATBManaged(toolID string) bool {
	receipt, ok := s.Tool(toolID)
	if !ok {
		return false
	}

	return receipt.Ownership == OwnershipManaged
}

// Remove deletes a tool receipt from state.
func (s *State) Remove(toolID string) {
	if len(s.Tools) == 0 {
		return
	}

	delete(s.Tools, toolID)
}

func (s *State) ensureInitialized() {
	if s.Version == 0 {
		s.Version = 1
	}

	if s.Tools == nil {
		s.Tools = make(map[string]ToolState)
	}
}

func loadFromPath(path string) (State, error) {
	//nolint:gosec // The state path comes from atb configuration or controlled tests.
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return State{Version: 1, Tools: map[string]ToolState{}}, nil
	default:
		return State{}, fmt.Errorf("read state file: %w", err)
	}

	var stateData State
	if err := json.Unmarshal(data, &stateData); err != nil {
		return State{Version: 1, Tools: map[string]ToolState{}}, errors.Join(
			ErrCorruptState,
			fmt.Errorf("decode state json: %w", err),
		)
	}

	if stateData.Version == 0 {
		stateData.Version = 1
	}

	if stateData.Tools == nil {
		stateData.Tools = make(map[string]ToolState)
	}

	return stateData, nil
}

func saveToPath(path string, st State) error {
	st.ensureInitialized()
	st.LastRunAt = time.Now().UTC()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), "state-*.json")
	if err != nil {
		return fmt.Errorf("create temp state file: %w", err)
	}

	tempPath := tempFile.Name()
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)

		return fmt.Errorf("write temp state file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf("close temp state file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)

		return fmt.Errorf("rename temp state file: %w", err)
	}

	return nil
}
