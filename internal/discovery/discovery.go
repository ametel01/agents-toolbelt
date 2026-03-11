// Package discovery reconciles PATH lookups with persisted tool state.
package discovery

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/state"
	"github.com/ametel01/agents-toolbelt/internal/verify"
)

const (
	// OwnershipManaged identifies tools installed through atb.
	OwnershipManaged = "managed"
	// OwnershipExternal identifies tools found on PATH without an atb receipt.
	OwnershipExternal = "external"
	// OwnershipMissing identifies tools not found on PATH and without any receipt.
	OwnershipMissing = "missing"
)

// LookPather abstracts executable lookup for tests.
type LookPather func(string) (string, error)

// ToolPresence captures the reconciled runtime view of a tool.
type ToolPresence struct {
	Tool         catalog.Tool
	Path         string
	Installed    bool
	Ownership    string
	Receipt      *state.ToolState
	VerifyResult *verify.VerifyResult
}

// Snapshot contains reconciled tool state keyed by tool ID.
type Snapshot struct {
	Tools map[string]ToolPresence
}

// ScanPATH looks up each catalog tool binary on PATH.
func ScanPATH(tools []catalog.Tool, lookPath LookPather) (map[string]string, error) {
	paths := make(map[string]string, len(tools))

	for _, tool := range tools {
		path, err := lookPath(tool.Bin)
		switch {
		case err == nil:
			paths[tool.ID] = path
		case errors.Is(err, exec.ErrNotFound):
			continue
		default:
			return nil, fmt.Errorf("look up %q on PATH: %w", tool.Bin, err)
		}
	}

	return paths, nil
}

// Reconcile merges catalog metadata, PATH discoveries, and persisted state.
func Reconcile(reg catalog.Registry, st state.State, paths map[string]string) Snapshot {
	snapshot := Snapshot{
		Tools: make(map[string]ToolPresence, len(reg.Tools())),
	}

	for _, tool := range reg.Tools() {
		path := paths[tool.ID]
		receipt, ok := st.Tool(tool.ID)

		presence := ToolPresence{
			Tool:      tool,
			Path:      path,
			Installed: path != "",
			Ownership: OwnershipMissing,
			Receipt:   receipt,
		}

		if ok {
			presence.Ownership = receipt.Ownership
		}

		if presence.Path != "" && !ok {
			presence.Ownership = OwnershipExternal
		}

		snapshot.Tools[tool.ID] = presence
	}

	return snapshot
}
