// Package plan builds install, update, and uninstall action plans from runtime state.
package plan

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
)

const (
	// ActionAlreadyInstalled marks a tool that does not require an install action.
	ActionAlreadyInstalled ActionType = "already_installed"
	// ActionInstall marks a tool that should be installed.
	ActionInstall ActionType = "install"
	// ActionSkip marks a tool that cannot be processed and should be reported.
	ActionSkip ActionType = "skip"
	// ActionUninstall marks a tool that should be uninstalled.
	ActionUninstall ActionType = "uninstall"
	// ActionUpdate marks a tool that should be updated.
	ActionUpdate ActionType = "update"
)

var errToolSelectionRequired = errors.New("at least one tool must be selected")

// ActionType identifies what should happen to a tool in a plan.
type ActionType string

// Action describes one planned operation for a tool.
type Action struct {
	Tool    catalog.Tool
	Type    ActionType
	Method  catalog.InstallMethod
	Manager pkgmgr.Manager
	Reason  string
}

// Plan is an ordered list of tool actions.
type Plan struct {
	Actions []Action
}

// BuildInstallPlan creates an ordered install plan for the selected tools.
func BuildInstallPlan(selected []catalog.Tool, snapshot discovery.Snapshot, managers []pkgmgr.Manager) (Plan, error) {
	if len(selected) == 0 {
		return Plan{}, errToolSelectionRequired
	}

	ordered := slices.Clone(selected)
	slices.SortStableFunc(ordered, func(left, right catalog.Tool) int {
		return compareTier(left.Tier, right.Tier)
	})

	actions := make([]Action, 0, len(ordered))
	for _, tool := range ordered {
		presence, found := snapshot.Tools[tool.ID]
		if found && presence.Ownership == discovery.OwnershipManaged {
			actions = append(actions, Action{
				Tool:   tool,
				Type:   ActionAlreadyInstalled,
				Reason: "already installed and managed by atb",
			})

			continue
		}

		if found && presence.Ownership == discovery.OwnershipExternal {
			actions = append(actions, Action{
				Tool:   tool,
				Type:   ActionAlreadyInstalled,
				Reason: "already installed externally",
			})

			continue
		}

		method, manager, err := pkgmgr.SelectBest(tool, managers)
		if err != nil {
			actions = append(actions, Action{
				Tool:   tool,
				Type:   ActionSkip,
				Reason: skipReason(tool, err),
			})

			continue
		}

		actions = append(actions, Action{
			Tool:    tool,
			Type:    ActionInstall,
			Method:  method,
			Manager: manager,
		})
	}

	return Plan{Actions: actions}, nil
}

func skipReason(tool catalog.Tool, err error) string {
	if !errors.Is(err, pkgmgr.ErrNoMatchingMethod) || len(tool.InstallMethods) == 0 {
		return err.Error()
	}

	managers := make([]string, 0, len(tool.InstallMethods))
	for _, method := range tool.InstallMethods {
		managers = append(managers, method.Manager)
	}

	return fmt.Sprintf("requires %s", strings.Join(managers, " or "))
}

func compareTier(left, right catalog.Tier) int {
	return tierRank(left) - tierRank(right)
}

func tierRank(tier catalog.Tier) int {
	switch tier {
	case catalog.TierMust:
		return 0
	case catalog.TierShould:
		return 1
	default:
		return 2
	}
}
