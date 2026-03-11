// Package state stores persistent tool ownership and verification metadata.
package state

import "time"

// State represents the persisted atb state file contents.
type State struct {
	Version   int
	Tools     map[string]ToolState
	LastRunAt time.Time
}

// ToolState records ownership and verification data for one tool.
type ToolState struct {
	ToolID          string
	Bin             string
	Ownership       string
	BinaryPath      string
	LastVerifyAt    time.Time
	LastVerifyOK    bool
	LastVerifyError string
	Version         string
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
