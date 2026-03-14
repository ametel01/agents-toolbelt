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

var errUninstallTargetsRequired = errors.New("provide at least one tool or use --all")

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
			if _, ok := resolveSelector(snapshot, id); !ok {
				return Plan{}, fmt.Errorf("unknown tool %q", id)
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

		method, manager, ok := methodForReceipt(presence.Tool, presence.Receipt.InstallManager, managers)
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

func methodForReceipt(tool catalog.Tool, installManager string, managers []pkgmgr.Manager) (catalog.InstallMethod, pkgmgr.Manager, bool) {
	for _, manager := range managers {
		if manager.Name() != installManager {
			continue
		}

		for _, method := range tool.InstallMethods {
			if method.Manager == installManager {
				return method, manager, true
			}
		}
	}

	return catalog.InstallMethod{}, nil, false
}

// resolveSelector checks whether selector matches any tool in the snapshot by
// ID or binary name, returning the canonical tool ID.
func resolveSelector(snapshot discovery.Snapshot, selector string) (string, bool) {
	if _, ok := snapshot.Tools[selector]; ok {
		return selector, true
	}

	for _, presence := range snapshot.Tools {
		if presence.Tool.Bin == selector {
			return presence.Tool.ID, true
		}
	}

	return "", false
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
