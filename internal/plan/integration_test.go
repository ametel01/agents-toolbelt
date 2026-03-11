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
	selected := mustSelectTools(t, registry, "fzf", "bat", "jq", "yq", "direnv")

	installPlan, err := BuildInstallPlan(selected, discovery.Snapshot{Tools: map[string]discovery.ToolPresence{}}, []pkgmgr.Manager{
		fakeManager{name: "brew"},
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan() error = %v", err)
	}

	st := state.State{}
	verifier := fakeVerifier{
		results: map[string]verify.VerifyResult{
			"fzf":    {ToolID: "fzf", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"bat":    {ToolID: "bat", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"jq":     {ToolID: "jq", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"yq":     {ToolID: "yq", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"direnv": {ToolID: "direnv", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
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
		"fzf":    "/usr/bin/fzf",
		"bat":    "/usr/bin/bat",
		"jq":     "/usr/bin/jq",
		"yq":     "/usr/bin/yq",
		"direnv": "/usr/bin/direnv",
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

	uninstallPlan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"fzf"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	uninstallSummary, err := ExecuteUninstallPlan(context.Background(), uninstallPlan, &st)
	if err != nil {
		t.Fatalf("ExecuteUninstallPlan() error = %v", err)
	}

	if len(uninstallSummary.Uninstalled) != 1 || uninstallSummary.Uninstalled[0] != "fzf" {
		t.Fatalf("uninstallSummary.Uninstalled = %#v, want [\"fzf\"]", uninstallSummary.Uninstalled)
	}
}

func TestPartialFailureAndVerifyFailure(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	selected := mustSelectTools(t, registry, "fzf", "bat", "jq")

	manager := fakeLifecycleManager{
		installErrs: map[string]error{"bat": errInstallFailed},
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
			"fzf": {ToolID: "fzf", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"jq":  {ToolID: "jq", Found: true, Verified: false, Error: "timed out", CheckedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteInstallPlan() error = %v", err)
	}

	if len(summary.Installed) != 1 || summary.Installed[0] != "fzf" {
		t.Fatalf("summary.Installed = %#v, want [\"fzf\"]", summary.Installed)
	}

	if len(summary.Failed) != 2 {
		t.Fatalf("len(summary.Failed) = %d, want 2", len(summary.Failed))
	}

	if !st.IsATBManaged("jq") {
		t.Fatal("managed receipt should persist when verification fails")
	}
}

func TestExternalToolNotUninstallable(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	fzf := mustTool(t, registry, "fzf")
	snapshot := discovery.Snapshot{
		Tools: map[string]discovery.ToolPresence{
			"fzf": {
				Tool:      fzf,
				Installed: true,
				Ownership: state.OwnershipExternal,
			},
		},
	}

	uninstallPlan, err := BuildUninstallPlan(snapshot, []pkgmgr.Manager{fakeManager{name: "brew"}}, []string{"fzf"}, false)
	if err != nil {
		t.Fatalf("BuildUninstallPlan() error = %v", err)
	}

	if uninstallPlan.Actions[0].Type != ActionSkip {
		t.Fatalf("uninstallPlan.Actions[0].Type = %q, want %q", uninstallPlan.Actions[0].Type, ActionSkip)
	}
}
