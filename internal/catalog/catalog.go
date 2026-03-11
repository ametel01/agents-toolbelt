// Package catalog provides the embedded tool registry used by atb.
package catalog

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Tier identifies a tool's default installation priority.
type Tier string

const (
	// TierMust marks tools that represent the recommended baseline.
	TierMust Tier = "must"
	// TierShould marks optional but recommended tools.
	TierShould Tier = "should"
	// TierNice marks optional extras hidden behind the expanded picker view.
	TierNice Tier = "nice"
)

const (
	shellHookNone     = "none"
	shellHookOptional = "optional"
	shellHookRequired = "required"
	authNone          = "none"
	authOptional      = "optional"
	authRequired      = "required"
)

var (
	errEmptyToolID          = errors.New("tool id is required")
	errEmptyToolBin         = errors.New("tool bin is required")
	errEmptyToolName        = errors.New("tool name is required")
	errInvalidTier          = errors.New("invalid tool tier")
	errEmptyInstallMethods  = errors.New("at least one install method is required")
	errEmptyVerifyCommand   = errors.New("verify command is required")
	errInvalidShellHook     = errors.New("invalid shell hook")
	errInvalidAuthMode      = errors.New("invalid auth mode")
	errEmptyInstallManager  = errors.New("install manager is required")
	errEmptyInstallPackage  = errors.New("install package is required")
	errEmptyInstallCommand  = errors.New("install command is required")
	errEmptyUpdateCommand   = errors.New("update command is required")
	errEmptyUninstallAction = errors.New("uninstall command is required")
	errDuplicateToolID      = errors.New("duplicate tool id")
	errDuplicateToolBin     = errors.New("duplicate tool bin")
)

//go:embed registry.json
var registryFile []byte

// Tool describes one installable CLI utility.
type Tool struct {
	ID              string          `json:"id"`
	Bin             string          `json:"bin"`
	Name            string          `json:"name"`
	Tier            Tier            `json:"tier"`
	Category        string          `json:"category"`
	Description     string          `json:"description"`
	Platforms       []string        `json:"platforms"`
	InstallMethods  []InstallMethod `json:"install_methods"`
	ShellHook       string          `json:"shell_hook"`
	Auth            string          `json:"auth"`
	ServiceDep      string          `json:"service_dependency"`
	Interactive     bool            `json:"interactive"`
	TUI             bool            `json:"tui"`
	Verify          VerifySpec      `json:"verify"`
	SkillExpose     bool            `json:"skill_expose"`
	DefaultSelected bool            `json:"default_selected"`
	Tags            []string        `json:"tags"`
}

// InstallMethod defines how to manage a tool with a package manager.
type InstallMethod struct {
	Manager          string   `json:"manager"`
	Package          string   `json:"package"`
	Command          []string `json:"command"`
	UpdateCommand    []string `json:"update_command"`
	UninstallCommand []string `json:"uninstall_command"`
	RequiresSudo     bool     `json:"requires_sudo"`
	TimeoutSeconds   int      `json:"timeout_seconds"`
}

// VerifySpec defines how to verify an installed tool.
type VerifySpec struct {
	Command           []string `json:"command"`
	TimeoutSeconds    int      `json:"timeout_seconds"`
	ExpectedExitCodes []int    `json:"expected_exit_codes"`
	VersionRegex      string   `json:"version_regex,omitempty"`
}

// Registry contains the embedded tool inventory.
type Registry struct {
	tools []Tool
}

// LoadRegistry parses and validates the embedded registry.
func LoadRegistry() (Registry, error) {
	var tools []Tool
	if err := json.Unmarshal(registryFile, &tools); err != nil {
		return Registry{}, fmt.Errorf("decode registry json: %w", err)
	}

	registry := Registry{tools: slices.Clone(tools)}
	if err := registry.Validate(); err != nil {
		return Registry{}, fmt.Errorf("validate registry: %w", err)
	}

	return registry, nil
}

// Validate ensures the registry is internally consistent.
func (r Registry) Validate() error {
	ids := make(map[string]struct{}, len(r.tools))
	bins := make(map[string]struct{}, len(r.tools))

	for _, tool := range r.tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("validate tool %q: %w", tool.ID, err)
		}

		if _, exists := ids[tool.ID]; exists {
			return fmt.Errorf("%w: %s", errDuplicateToolID, tool.ID)
		}

		if _, exists := bins[tool.Bin]; exists {
			return fmt.Errorf("%w: %s", errDuplicateToolBin, tool.Bin)
		}

		ids[tool.ID] = struct{}{}
		bins[tool.Bin] = struct{}{}
	}

	return nil
}

// Tools returns a shallow copy of the registry contents.
func (r Registry) Tools() []Tool {
	return slices.Clone(r.tools)
}

// ByID returns a tool by ID.
func (r Registry) ByID(id string) (Tool, bool) {
	for _, tool := range r.tools {
		if tool.ID == id {
			return tool, true
		}
	}

	return Tool{}, false
}

// ByTier returns all tools for a tier.
func (r Registry) ByTier(tier Tier) []Tool {
	return r.filter(func(tool Tool) bool {
		return tool.Tier == tier
	})
}

// ByCategory returns all tools in a category.
func (r Registry) ByCategory(category string) []Tool {
	return r.filter(func(tool Tool) bool {
		return tool.Category == category
	})
}

// ForPlatform returns all tools available on a platform.
func (r Registry) ForPlatform(platform string) []Tool {
	return r.filter(func(tool Tool) bool {
		return slices.Contains(tool.Platforms, platform)
	})
}

func (r Registry) filter(predicate func(Tool) bool) []Tool {
	filtered := make([]Tool, 0, len(r.tools))

	for _, tool := range r.tools {
		if predicate(tool) {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}

// Validate ensures a tool has the required fields and supported values.
func (t Tool) Validate() error {
	switch {
	case t.ID == "":
		return errEmptyToolID
	case t.Bin == "":
		return errEmptyToolBin
	case t.Name == "":
		return errEmptyToolName
	}

	if !isValidTier(t.Tier) {
		return fmt.Errorf("%w: %s", errInvalidTier, t.Tier)
	}

	if !isValidShellHook(t.ShellHook) {
		return fmt.Errorf("%w: %s", errInvalidShellHook, t.ShellHook)
	}

	if !isValidAuthMode(t.Auth) {
		return fmt.Errorf("%w: %s", errInvalidAuthMode, t.Auth)
	}

	if len(t.InstallMethods) == 0 {
		return errEmptyInstallMethods
	}

	for _, method := range t.InstallMethods {
		if err := method.Validate(); err != nil {
			return fmt.Errorf("validate install method %q: %w", method.Manager, err)
		}
	}

	return t.Verify.Validate()
}

// Validate ensures an install method is usable.
func (m InstallMethod) Validate() error {
	switch {
	case m.Manager == "":
		return errEmptyInstallManager
	case m.Package == "":
		return errEmptyInstallPackage
	case len(m.Command) == 0:
		return errEmptyInstallCommand
	case len(m.UpdateCommand) == 0:
		return errEmptyUpdateCommand
	case len(m.UninstallCommand) == 0:
		return errEmptyUninstallAction
	default:
		return nil
	}
}

// Validate ensures a verify spec is usable.
func (v VerifySpec) Validate() error {
	if len(v.Command) == 0 {
		return errEmptyVerifyCommand
	}

	return nil
}

func isValidTier(tier Tier) bool {
	switch tier {
	case TierMust, TierShould, TierNice:
		return true
	default:
		return false
	}
}

func isValidShellHook(shellHook string) bool {
	switch shellHook {
	case shellHookNone, shellHookOptional, shellHookRequired:
		return true
	default:
		return false
	}
}

func isValidAuthMode(authMode string) bool {
	switch authMode {
	case authNone, authOptional, authRequired:
		return true
	default:
		return false
	}
}
