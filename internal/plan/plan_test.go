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
	selected := mustSelectTools(t, registry, "terraform", "rg", "uv")

	plan, err := BuildInstallPlan(selected, discovery.Snapshot{Tools: map[string]discovery.ToolPresence{}}, []pkgmgr.Manager{
		fakeManager{name: "brew"},
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan() error = %v", err)
	}

	if len(plan.Actions) != 3 {
		t.Fatalf("len(plan.Actions) = %d, want 3", len(plan.Actions))
	}

	if plan.Actions[0].Tool.ID != "rg" || plan.Actions[1].Tool.ID != "uv" || plan.Actions[2].Tool.ID != "terraform" {
		t.Fatalf("plan.Actions order = [%s %s %s], want [rg uv terraform]",
			plan.Actions[0].Tool.ID,
			plan.Actions[1].Tool.ID,
			plan.Actions[2].Tool.ID,
		)
	}
}

func TestBuildInstallPlanAlreadyInstalled(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	selected := mustSelectTools(t, registry, "rg", "jq")
	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      selected[0],
				Installed: true,
				Ownership: discovery.OwnershipExternal,
			},
			"jq": {
				Tool:      selected[1],
				Installed: true,
				Ownership: discovery.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:    "jq",
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
	selected := mustSelectTools(t, registry, "rg")

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
	rg := mustTool(t, registry, "rg")
	uv := mustTool(t, registry, "uv")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "rg",
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

	if plan.Actions[0].Tool.ID != "rg" || plan.Actions[0].Type != ActionUpdate {
		t.Fatalf("plan.Actions[0] = %#v, want rg update action", plan.Actions[0])
	}
}

func TestBuildUpdatePlanUsesRecordedInstallManager(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	rg := mustTool(t, registry, "rg")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "rg",
					Ownership:      state.OwnershipManaged,
					InstallManager: "apt",
				},
			},
		},
	}

	plan, err := BuildUpdatePlan(snapshot, []pkgmgr.Manager{
		fakeManager{name: "brew"},
		fakeManager{name: "apt"},
	}, "")
	if err != nil {
		t.Fatalf("BuildUpdatePlan() error = %v", err)
	}

	if len(plan.Actions) != 1 {
		t.Fatalf("len(plan.Actions) = %d, want 1", len(plan.Actions))
	}

	if plan.Actions[0].Method.Manager != "apt" {
		t.Fatalf("plan.Actions[0].Method.Manager = %q, want %q", plan.Actions[0].Method.Manager, "apt")
	}
}

func TestBuildUninstallPlanRefusesExternalTools(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	rg := mustTool(t, registry, "rg")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Ownership: state.OwnershipExternal,
			},
		},
	}

	plan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"rg"}, false)
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
	rg := mustTool(t, registry, "rg")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "rg",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
				},
			},
		},
	}

	plan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"rg"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	if plan.Actions[0].Type != ActionUninstall {
		t.Fatalf("plan.Actions[0].Type = %q, want %q", plan.Actions[0].Type, ActionUninstall)
	}
}

func TestBuildUpdatePlanUnknownToolReturnsError(t *testing.T) {
	t.Parallel()

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{},
	}

	_, err := BuildUpdatePlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, "does-not-exist")
	if err == nil {
		t.Fatal("BuildUpdatePlan() expected error for unknown tool, got nil")
	}
}

func TestBuildUpdatePlanExternalToolReturnsSkip(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tool := mustTool(t, registry, "jq")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"jq": {
				Tool:      tool,
				Installed: true,
				Ownership: state.OwnershipExternal,
			},
		},
	}

	plan, err := BuildUpdatePlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, "jq")
	if err != nil {
		t.Fatalf("BuildUpdatePlan() error = %v", err)
	}

	if len(plan.Actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1", len(plan.Actions))
	}

	if plan.Actions[0].Type != ActionSkip {
		t.Fatalf("action type = %q, want %q", plan.Actions[0].Type, ActionSkip)
	}

	if plan.Actions[0].Reason != "tool is not managed by atb" {
		t.Fatalf("action reason = %q, want %q", plan.Actions[0].Reason, "tool is not managed by atb")
	}
}

func TestBuildUpdatePlanMatchesByBinaryName(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tool := mustTool(t, registry, "difftastic") // bin = "difft"

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"difftastic": {
				Tool:      tool,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "difftastic",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
				},
			},
		},
	}

	plan, err := BuildUpdatePlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, "difft")
	if err != nil {
		t.Fatalf("BuildUpdatePlan() error = %v", err)
	}

	if len(plan.Actions) != 1 || plan.Actions[0].Tool.ID != "difftastic" {
		t.Fatalf("expected 1 action for difftastic, got %d actions", len(plan.Actions))
	}
}

func TestBuildUninstallPlanUnknownToolReturnsError(t *testing.T) {
	t.Parallel()

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{},
	}

	_, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"does-not-exist"}, false)
	if err == nil {
		t.Fatal("BuildUninstallPlan() expected error for unknown tool, got nil")
	}
}

func TestBuildUpdatePlanPrefersReceiptCommands(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	rg := mustTool(t, registry, "rg")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "rg",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
					InstallCommand: []string{"brew", "install", "ripgrep-custom"},
					UpdateCommand:  []string{"brew", "upgrade", "ripgrep-custom"},
				},
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

	if plan.Actions[0].Method.UpdateCommand[2] != "ripgrep-custom" {
		t.Fatalf("plan.Actions[0].Method.UpdateCommand = %v, want receipt-stored command", plan.Actions[0].Method.UpdateCommand)
	}
}

func TestBuildUninstallPlanPrefersReceiptCommands(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	rg := mustTool(t, registry, "rg")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:           "rg",
					Ownership:        state.OwnershipManaged,
					InstallManager:   "brew",
					InstallCommand:   []string{"brew", "install", "ripgrep-custom"},
					UninstallCommand: []string{"brew", "uninstall", "ripgrep-custom"},
				},
			},
		},
	}

	plan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"rg"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	if len(plan.Actions) != 1 {
		t.Fatalf("len(plan.Actions) = %d, want 1", len(plan.Actions))
	}

	if plan.Actions[0].Method.UninstallCommand[2] != "ripgrep-custom" {
		t.Fatalf("plan.Actions[0].Method.UninstallCommand = %v, want receipt-stored command", plan.Actions[0].Method.UninstallCommand)
	}
}

func TestBuildUpdatePlanFallsToCatalogWithoutReceiptCommands(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	rg := mustTool(t, registry, "rg")

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "rg",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
					// No stored commands — should fall back to catalog.
				},
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

	// Should fall back to catalog method.
	if plan.Actions[0].Method.Manager != "brew" {
		t.Fatalf("plan.Actions[0].Method.Manager = %q, want %q", plan.Actions[0].Method.Manager, "brew")
	}
}

func TestBuildUninstallPlanMatchesByBinaryName(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tool := mustTool(t, registry, "difftastic") // bin = "difft"

	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"difftastic": {
				Tool:      tool,
				Ownership: state.OwnershipManaged,
				Receipt: &state.ToolState{
					ToolID:         "difftastic",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
				},
			},
		},
	}

	plan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"difft"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	if len(plan.Actions) != 1 || plan.Actions[0].Tool.ID != "difftastic" {
		t.Fatalf("expected 1 uninstall action for difftastic, got %d actions", len(plan.Actions))
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
