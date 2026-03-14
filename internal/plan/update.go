package plan

import (
	"fmt"

	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/state"
)

// BuildUpdatePlan creates an update plan for atb-managed tools.
func BuildUpdatePlan(snapshot discovery.Snapshot, managers []pkgmgr.Manager, toolID string) (Plan, error) {
	if toolID != "" && !resolveSelector(snapshot, toolID) {
		return Plan{}, fmt.Errorf("%w: %s", errUnknownTool, toolID)
	}

	actions := make([]Action, 0, len(snapshot.Tools))

	for _, presence := range snapshot.Tools {
		if !shouldPlanTool(presence, toolID) {
			continue
		}

		if presence.Ownership != state.OwnershipManaged || presence.Receipt == nil {
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
			Type:    ActionUpdate,
			Method:  method,
			Manager: manager,
		})
	}

	sortActions(actions)

	return Plan{Actions: actions}, nil
}
