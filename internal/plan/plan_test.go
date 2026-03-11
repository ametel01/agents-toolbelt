package plan

import (
	"context"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/state"
)

func TestBuildInstallPlanOrdersByTier(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	selected := mustSelectTools(t, registry, "k9s", "fzf", "starship")

	plan, err := BuildInstallPlan(selected, discovery.Snapshot{Tools: map[string]discovery.ToolPresence{}}, []pkgmgr.Manager{
		fakeManager{name: "brew"},
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan() error = %v", err)
	}

	if len(plan.Actions) != 3 {
		t.Fatalf("len(plan.Actions) = %d, want 3", len(plan.Actions))
	}

	if plan.Actions[0].Tool.ID != "fzf" || plan.Actions[1].Tool.ID != "starship" || plan.Actions[2].Tool.ID != "k9s" {
		t.Fatalf("plan.Actions order = [%s %s %s], want [fzf starship k9s]",
			plan.Actions[0].Tool.ID,
			plan.Actions[1].Tool.ID,
			plan.Actions[2].Tool.ID,
		)
	}
}

func TestBuildInstallPlanAlreadyInstalled(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	selected := mustSelectTools(t, registry, "fzf", "bat")
	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"fzf": {
				Tool:      selected[0],
				Installed: true,
				Ownership: discovery.OwnershipExternal,
			},
			"bat": {
				Tool:      selected[1],
				Installed: true,
				Ownership: discovery.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:    "bat",
					Ownership: state.OwnershipManaged,
				},
			},
		},
	}

	plan, err := BuildInstallPlan(selected, snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}})
	if err != nil {
		t.Fatalf("BuildInstallPlan() error = %v", err)
	}

	for _, action := range plan.Actions {
		if action.Type != ActionAlreadyInstalled {
			t.Fatalf("action.Type = %q, want %q", action.Type, ActionAlreadyInstalled)
		}
	}
}

func TestBuildInstallPlanNoMatchingManager(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	selected := mustSelectTools(t, registry, "fzf")

	plan, err := BuildInstallPlan(selected, discovery.Snapshot{Tools: map[string]discovery.ToolPresence{}}, []pkgmgr.Manager{
		fakeManager{name: "pipx"},
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan() error = %v", err)
	}

	if plan.Actions[0].Type != ActionSkip {
		t.Fatalf("plan.Actions[0].Type = %q, want %q", plan.Actions[0].Type, ActionSkip)
	}
}

func TestBuildUpdatePlanManagedOnly(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	fzf := mustTool(t, registry, "fzf")
	uv := mustTool(t, registry, "uv")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"fzf": {
				Tool:      fzf,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "fzf",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
				},
			},
			"uv": {
				Tool:      uv,
				Ownership: state.OwnershipExternal,
			},
		},
	}

	plan, err := BuildUpdatePlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, "")
	if err != nil {
		t.Fatalf("BuildUpdatePlan() error = %v", err)
	}

	if len(plan.Actions) != 1 {
		t.Fatalf("len(plan.Actions) = %d, want 1", len(plan.Actions))
	}

	if plan.Actions[0].Tool.ID != "fzf" || plan.Actions[0].Type != ActionUpdate {
		t.Fatalf("plan.Actions[0] = %#v, want fzf update action", plan.Actions[0])
	}
}

func TestBuildUninstallPlanRefusesExternalTools(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	fzf := mustTool(t, registry, "fzf")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"fzf": {
				Tool:      fzf,
				Ownership: state.OwnershipExternal,
			},
		},
	}

	plan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"fzf"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	if plan.Actions[0].Type != ActionSkip {
		t.Fatalf("plan.Actions[0].Type = %q, want %q", plan.Actions[0].Type, ActionSkip)
	}
}

func TestBuildUninstallPlanManagedTool(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	fzf := mustTool(t, registry, "fzf")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"fzf": {
				Tool:      fzf,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "fzf",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
				},
			},
		},
	}

	plan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"fzf"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	if plan.Actions[0].Type != ActionUninstall {
		t.Fatalf("plan.Actions[0].Type = %q, want %q", plan.Actions[0].Type, ActionUninstall)
	}
}

func mustLoadRegistry(t *testing.T) catalog.Registry {
	t.Helper()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	return registry
}

func mustSelectTools(t *testing.T, registry catalog.Registry, ids ...string) []catalog.Tool {
	t.Helper()

	tools := make([]catalog.Tool, 0, len(ids))
	for _, id := range ids {
		tools = append(tools, mustTool(t, registry, id))
	}

	return tools
}

func mustTool(t *testing.T, registry catalog.Registry, id string) catalog.Tool {
	t.Helper()

	tool, ok := registry.ByID(id)
	if !ok {
		t.Fatalf("registry.ByID(%q) did not find a tool", id)
	}

	return tool
}

type fakeManager struct {
	name string
}

func (f fakeManager) Name() string {
	return f.name
}

func (f fakeManager) Available() bool {
	return true
}

func (f fakeManager) Install(_ context.Context, _ catalog.InstallMethod) error {
	return nil
}

func (f fakeManager) Update(_ context.Context, _ catalog.InstallMethod) error {
	return nil
}

func (f fakeManager) Uninstall(_ context.Context, _ catalog.InstallMethod) error {
	return nil
}
