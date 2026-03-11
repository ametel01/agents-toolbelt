package plan

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/state"
	"github.com/ametel01/agents-toolbelt/internal/verify"
)

var errInstallFailed = errors.New("install failed")

func TestExecuteInstallPlanHappyPath(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	plan := Plan{
		Actions: []Action{
			installAction(mustTool(t, registry, "rg")),
			installAction(mustTool(t, registry, "yq")),
			installAction(mustTool(t, registry, "jq")),
		},
	}

	st := state.State{}
	summary, err := ExecuteInstallPlan(context.Background(), plan, &st, fakeVerifier{
		results: map[string]verify.VerifyResult{
			"rg": {ToolID: "rg", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"yq": {ToolID: "yq", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"jq": {ToolID: "jq", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteInstallPlan() error = %v", err)
	}

	if len(summary.Installed) != 3 {
		t.Fatalf("len(summary.Installed) = %d, want 3", len(summary.Installed))
	}

	if !st.IsATBManaged("rg") || !st.IsATBManaged("yq") || !st.IsATBManaged("jq") {
		t.Fatal("managed receipts were not persisted for installed tools")
	}
}

func TestExecuteInstallPlanContinuesAfterFailure(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	manager := fakeLifecycleManager{
		installErrs: map[string]error{"yq": errInstallFailed},
	}

	plan := Plan{
		Actions: []Action{
			installActionWithManager(mustTool(t, registry, "rg"), manager),
			installActionWithManager(mustTool(t, registry, "yq"), manager),
		},
	}

	st := state.State{}
	summary, err := ExecuteInstallPlan(context.Background(), plan, &st, fakeVerifier{
		results: map[string]verify.VerifyResult{
			"rg": {ToolID: "rg", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteInstallPlan() error = %v", err)
	}

	if len(summary.Failed) != 1 || summary.Failed[0].ToolID != "yq" {
		t.Fatalf("summary.Failed = %#v, want yq install failure", summary.Failed)
	}

	if !st.IsATBManaged("rg") || st.IsATBManaged("yq") {
		t.Fatal("install receipts do not match expected success and failure outcomes")
	}
}

func TestExecuteInstallPlanVerifyFailureKeepsReceipt(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tool := mustTool(t, registry, "rg")

	st := state.State{}
	summary, err := ExecuteInstallPlan(context.Background(), Plan{
		Actions: []Action{installAction(tool)},
	}, &st, fakeVerifier{
		results: map[string]verify.VerifyResult{
			"rg": {ToolID: "rg", Found: true, Verified: false, Error: "timed out", CheckedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteInstallPlan() error = %v", err)
	}

	if len(summary.Failed) != 1 {
		t.Fatalf("len(summary.Failed) = %d, want 1", len(summary.Failed))
	}

	if !st.IsATBManaged("rg") {
		t.Fatal("verification failure should keep a managed receipt")
	}
}

func TestExecuteUpdatePlanManagedOnly(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tool := mustTool(t, registry, "rg")

	st := state.State{
		Version: 1,
		Tools: map[string]state.ToolState{
			"rg": {
				ToolID:         "rg",
				Bin:            "rg",
				Ownership:      state.OwnershipManaged,
				InstallManager: "brew",
			},
		},
	}

	summary, err := ExecuteUpdatePlan(context.Background(), Plan{
		Actions: []Action{{
			Tool:    tool,
			Type:    ActionUpdate,
			Method:  installMethodForTool(tool, "brew"),
			Manager: fakeLifecycleManager{},
		}},
	}, &st, fakeVerifier{
		results: map[string]verify.VerifyResult{
			"rg": {ToolID: "rg", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteUpdatePlan() error = %v", err)
	}

	if len(summary.Updated) != 1 || summary.Updated[0] != "rg" {
		t.Fatalf("summary.Updated = %#v, want [\"rg\"]", summary.Updated)
	}
}

func TestExecuteUninstallPlanRefusesExternalTools(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tool := mustTool(t, registry, "rg")

	summary, err := ExecuteUninstallPlan(context.Background(), Plan{
		Actions: []Action{{
			Tool:   tool,
			Type:   ActionSkip,
			Reason: "tool is not managed by atb",
		}},
	}, &state.State{})
	if err != nil {
		t.Fatalf("ExecuteUninstallPlan() error = %v", err)
	}

	if len(summary.External) != 1 || summary.External[0] != "rg" {
		t.Fatalf("summary.External = %#v, want [\"rg\"]", summary.External)
	}
}

func TestExecuteUninstallPlanRemovesReceipt(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tool := mustTool(t, registry, "rg")

	st := state.State{
		Version: 1,
		Tools: map[string]state.ToolState{
			"rg": {
				ToolID:         "rg",
				Bin:            "rg",
				Ownership:      state.OwnershipManaged,
				InstallManager: "brew",
			},
		},
	}

	summary, err := ExecuteUninstallPlan(context.Background(), Plan{
		Actions: []Action{{
			Tool:    tool,
			Type:    ActionUninstall,
			Method:  installMethodForTool(tool, "brew"),
			Manager: fakeLifecycleManager{},
		}},
	}, &st)
	if err != nil {
		t.Fatalf("ExecuteUninstallPlan() error = %v", err)
	}

	if len(summary.Uninstalled) != 1 || summary.Uninstalled[0] != "rg" {
		t.Fatalf("summary.Uninstalled = %#v, want [\"rg\"]", summary.Uninstalled)
	}

	if st.IsATBManaged("rg") {
		t.Fatal("managed receipt should be removed after uninstall")
	}
}

func installAction(tool catalog.Tool) Action {
	return installActionWithManager(tool, fakeLifecycleManager{})
}

func installActionWithManager(tool catalog.Tool, manager fakeLifecycleManager) Action {
	return Action{
		Tool:    tool,
		Type:    ActionInstall,
		Method:  installMethodForTool(tool, "brew"),
		Manager: manager,
	}
}

func installMethodForTool(tool catalog.Tool, managerName string) catalog.InstallMethod {
	for _, method := range tool.InstallMethods {
		if method.Manager == managerName {
			return method
		}
	}

	return tool.InstallMethods[0]
}

type fakeLifecycleManager struct {
	installErrs   map[string]error
	uninstallErrs map[string]error
	updateErrs    map[string]error
}

func (f fakeLifecycleManager) Name() string {
	return "brew"
}

func (f fakeLifecycleManager) Available() bool {
	return true
}

func (f fakeLifecycleManager) Install(_ context.Context, method catalog.InstallMethod) error {
	return f.installErrs[method.Package]
}

func (f fakeLifecycleManager) Update(_ context.Context, method catalog.InstallMethod) error {
	return f.updateErrs[method.Package]
}

func (f fakeLifecycleManager) Uninstall(_ context.Context, method catalog.InstallMethod) error {
	return f.uninstallErrs[method.Package]
}

type fakeVerifier struct {
	errs    map[string]error
	results map[string]verify.VerifyResult
}

func (f fakeVerifier) Check(_ context.Context, tool catalog.Tool) (verify.VerifyResult, error) {
	if err := f.errs[tool.ID]; err != nil {
		return verify.VerifyResult{}, err
	}

	if result, ok := f.results[tool.ID]; ok {
		return result, nil
	}

	return verify.VerifyResult{ToolID: tool.ID, Found: true, Verified: true, CheckedAt: time.Now().UTC()}, nil
}
