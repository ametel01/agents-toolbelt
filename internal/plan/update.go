package plan

import (
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/state"
)

// BuildUpdatePlan creates an update plan for atb-managed tools.
func BuildUpdatePlan(snapshot discovery.Snapshot, managers []pkgmgr.Manager, toolID string) (Plan, error) {
	actions := make([]Action, 0, len(snapshot.Tools))

	for _, presence := range snapshot.Tools {
		if !shouldPlanTool(presence.Tool.ID, toolID) {
			continue
		}

		if presence.Ownership != state.OwnershipManaged || presence.Receipt == nil {
			continue
		}

		method, manager, err := pkgmgr.SelectBest(presence.Tool, managers)
		if err != nil {
			actions = append(actions, Action{
				Tool:   presence.Tool,
				Type:   ActionSkip,
				Reason: err.Error(),
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
