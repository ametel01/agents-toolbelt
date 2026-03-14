package plan

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/state"
)

var (
	errUninstallTargetsRequired = errors.New("provide at least one tool or use --all")
	errUnknownTool              = errors.New("unknown tool")
)

// BuildUninstallPlan creates an uninstall plan for managed tools only.
func BuildUninstallPlan(
	snapshot discovery.Snapshot,
	managers []pkgmgr.Manager,
	toolIDs []string,
	uninstallAll bool,
) (Plan, error) {
	if !uninstallAll && len(toolIDs) == 0 {
		return Plan{}, errUninstallTargetsRequired
	}

	if !uninstallAll {
		for _, id := range toolIDs {
			if !resolveSelector(snapshot, id) {
				return Plan{}, fmt.Errorf("%w: %s", errUnknownTool, id)
			}
		}
	}

	actions := make([]Action, 0, len(snapshot.Tools))
	for _, presence := range snapshot.Tools {
		if !uninstallAll && !matchesSelector(presence, toolIDs) {
			continue
		}

		if presence.Ownership != state.OwnershipManaged || presence.Receipt == nil {
			actions = append(actions, Action{
				Tool:   presence.Tool,
				Type:   ActionSkip,
				Reason: "tool is not managed by atb",
			})

			continue
		}

		method, manager, ok := methodForReceipt(presence.Tool, presence.Receipt, managers)
		if !ok {
			actions = append(actions, Action{
				Tool:   presence.Tool,
				Type:   ActionSkip,
				Reason: "install manager is no longer available",
			})

			continue
		}

		actions = append(actions, Action{
			Tool:    presence.Tool,
			Type:    ActionUninstall,
			Method:  method,
			Manager: manager,
		})
	}

	sortActions(actions)

	return Plan{Actions: actions}, nil
}

func methodForReceipt(
	tool catalog.Tool,
	receipt *state.ToolState,
	managers []pkgmgr.Manager,
) (catalog.InstallMethod, pkgmgr.Manager, bool) {
	manager := findManager(receipt.InstallManager, managers)
	if manager == nil {
		return catalog.InstallMethod{}, nil, false
	}

	if hasStoredCommands(receipt) {
		return catalog.InstallMethod{
			Manager:          receipt.InstallManager,
			Package:          receipt.InstallPackage,
			Command:          receipt.InstallCommand,
			UpdateCommand:    receipt.UpdateCommand,
			UninstallCommand: receipt.UninstallCommand,
		}, manager, true
	}

	for _, method := range tool.InstallMethods {
		if method.Manager == receipt.InstallManager {
			return method, manager, true
		}
	}

	return catalog.InstallMethod{}, nil, false
}

func findManager(name string, managers []pkgmgr.Manager) pkgmgr.Manager {
	for _, m := range managers {
		if m.Name() == name {
			return m
		}
	}

	return nil
}

func hasStoredCommands(receipt *state.ToolState) bool {
	return len(receipt.InstallCommand) > 0 ||
		len(receipt.UpdateCommand) > 0 ||
		len(receipt.UninstallCommand) > 0
}

// resolveSelector checks whether selector matches any tool in the snapshot by
// ID or binary name.
func resolveSelector(snapshot discovery.Snapshot, selector string) bool {
	if _, ok := snapshot.Tools[selector]; ok {
		return true
	}

	for _, presence := range snapshot.Tools {
		if presence.Tool.Bin == selector {
			return true
		}
	}

	return false
}

func shouldPlanTool(presence discovery.ToolPresence, requested string) bool {
	return requested == "" || presence.Tool.ID == requested || presence.Tool.Bin == requested
}

func matchesSelector(presence discovery.ToolPresence, selectors []string) bool {
	for _, sel := range selectors {
		if presence.Tool.ID == sel || presence.Tool.Bin == sel {
			return true
		}
	}

	return false
}

func sortActions(actions []Action) {
	slices.SortStableFunc(actions, func(left, right Action) int {
		if rank := compareTier(left.Tool.Tier, right.Tool.Tier); rank != 0 {
			return rank
		}

		if left.Tool.ID < right.Tool.ID {
			return -1
		}

		if left.Tool.ID > right.Tool.ID {
			return 1
		}

		return 0
	})
}
