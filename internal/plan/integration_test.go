package plan

import (
	"context"
	"testing"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/state"
	"github.com/ametel01/agents-toolbelt/internal/verify"
)

func TestInstallUpdateUninstallCycle(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	selected := mustSelectTools(t, registry, "rg", "jq", "yq", "direnv", "just")

	installPlan, err := BuildInstallPlan(selected, discovery.Snapshot{Tools: map[string]discovery.ToolPresence{}}, []pkgmgr.Manager{
		fakeManager{name: "brew"},
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan() error = %v", err)
	}

	st := state.State{}
	verifier := fakeVerifier{
		results: map[string]verify.VerifyResult{
			"rg":     {ToolID: "rg", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"jq":     {ToolID: "jq", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"yq":     {ToolID: "yq", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"direnv": {ToolID: "direnv", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"just":   {ToolID: "just", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
		},
	}

	summary, err := ExecuteInstallPlan(context.Background(), installPlan, &st, verifier)
	if err != nil {
		t.Fatalf("ExecuteInstallPlan() error = %v", err)
	}

	if len(summary.Installed) != 5 {
		t.Fatalf("len(summary.Installed) = %d, want 5", len(summary.Installed))
	}

	snapshot := discovery.Reconcile(registry, st, map[string]string{
		"rg":     "/usr/bin/rg",
		"jq":     "/usr/bin/jq",
		"yq":     "/usr/bin/yq",
		"direnv": "/usr/bin/direnv",
		"just":   "/usr/bin/just",
	})

	updatePlan, err := BuildUpdatePlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, "")
	if err != nil {
		t.Fatalf("BuildUpdatePlan() error = %v", err)
	}

	updateSummary, err := ExecuteUpdatePlan(context.Background(), updatePlan, &st, verifier)
	if err != nil {
		t.Fatalf("ExecuteUpdatePlan() error = %v", err)
	}

	if len(updateSummary.Updated) != 5 {
		t.Fatalf("len(updateSummary.Updated) = %d, want 5", len(updateSummary.Updated))
	}

	uninstallPlan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"rg"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	uninstallSummary, err := ExecuteUninstallPlan(context.Background(), uninstallPlan, &st)
	if err != nil {
		t.Fatalf("ExecuteUninstallPlan() error = %v", err)
	}

	if len(uninstallSummary.Uninstalled) != 1 || uninstallSummary.Uninstalled[0] != "rg" {
		t.Fatalf("uninstallSummary.Uninstalled = %#v, want [\"rg\"]", uninstallSummary.Uninstalled)
	}
}

func TestPartialFailureAndVerifyFailure(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	selected := mustSelectTools(t, registry, "rg", "jq", "yq")

	manager := fakeLifecycleManager{
		installErrs: map[string]error{"jq": errInstallFailed},
	}

	installPlan, err := BuildInstallPlan(selected, discovery.Snapshot{Tools: map[string]discovery.ToolPresence{}}, []pkgmgr.Manager{
		manager,
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan() error = %v", err)
	}

	st := state.State{}
	summary, err := ExecuteInstallPlan(context.Background(), installPlan, &st, fakeVerifier{
		results: map[string]verify.VerifyResult{
			"rg": {ToolID: "rg", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"yq": {ToolID: "yq", Found: true, Verified: false, Error: "timed out", CheckedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteInstallPlan() error = %v", err)
	}

	if len(summary.Installed) != 1 || summary.Installed[0] != "rg" {
		t.Fatalf("summary.Installed = %#v, want [\"rg\"]", summary.Installed)
	}

	if len(summary.Failed) != 2 {
		t.Fatalf("len(summary.Failed) = %d, want 2", len(summary.Failed))
	}

	if !st.IsATBManaged("yq") {
		t.Fatal("managed receipt should persist when verification fails")
	}
}

func TestExternalToolNotUninstallable(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	rg := mustTool(t, registry, "rg")
	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"rg": {
				Tool:      rg,
				Installed: true,
				Ownership: state.OwnershipExternal,
			},
		},
	}

	uninstallPlan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"rg"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	if uninstallPlan.Actions[0].Type != ActionSkip {
		t.Fatalf("uninstallPlan.Actions[0].Type = %q, want %q", uninstallPlan.Actions[0].Type, ActionSkip)
	}
}
