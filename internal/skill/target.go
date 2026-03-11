package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// Target identifies an AI coding agent that can receive skill files.
type Target struct {
	ID   string
	Name string
	// RelPath is the path relative to the user's home directory.
	RelPath string
}

// Targets lists all supported skill targets.
var Targets = []Target{
	{ID: "claude", Name: "Claude Code", RelPath: filepath.Join(".claude", "skills", "cli-tools", "SKILL.md")},
	{ID: "codex", Name: "Codex", RelPath: filepath.Join(".agents", "skills", "cli-tools", "SKILL.md")},
}

// AllTargets returns a copy of every supported target.
func AllTargets() []Target {
	out := make([]Target, len(Targets))
	copy(out, Targets)

	return out
}

// PathsForTargets resolves the absolute file paths for the given targets.
func PathsForTargets(targets []Target) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home dir: %w", err)
	}

	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		paths = append(paths, filepath.Join(homeDir, target.RelPath))
	}

	return paths, nil
}

// DefaultPaths returns the standard cli-tools skill destinations for Claude Code and Codex.
func DefaultPaths() ([]string, error) {
	return PathsForTargets(Targets)
}
