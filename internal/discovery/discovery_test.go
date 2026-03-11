package discovery

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/state"
)

var (
	errUnexpectedBin = errors.New("unexpected bin")
	errLookupFailed  = errors.New("lookup failed")
)

func TestScanPATH(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tools := registry.ByTier(catalog.TierMust)[:2]

	paths, err := ScanPATH(tools, func(bin string) (string, error) {
		switch bin {
		case "gh":
			return "/usr/bin/gh", nil
		case "jq":
			return "", exec.ErrNotFound
		default:
			return "", errUnexpectedBin
		}
	})
	if err != nil {
		t.Fatalf("ScanPATH() error = %v", err)
	}

	if paths["gh"] != "/usr/bin/gh" {
		t.Fatalf("paths[\"gh\"] = %q, want %q", paths["gh"], "/usr/bin/gh")
	}

	if _, ok := paths["jq"]; ok {
		t.Fatal("paths should not include jq when it is not found on PATH")
	}
}

func TestScanPATHUnexpectedError(t *testing.T) {
	t.Parallel()

	tools := []catalog.Tool{{ID: "fzf", Bin: "fzf"}}

	_, err := ScanPATH(tools, func(string) (string, error) {
		return "", errLookupFailed
	})
	if err == nil {
		t.Fatal("ScanPATH() error = nil, want lookup failure")
	}
}

func TestReconcile(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	stateData := state.State{
		Tools: map[string]state.ToolState{
			"fzf": {
				ToolID:    "fzf",
				Bin:       "fzf",
				Ownership: OwnershipManaged,
			},
			"bat": {
				ToolID:    "bat",
				Bin:       "bat",
				Ownership: OwnershipManaged,
			},
		},
	}

	snapshot := Reconcile(registry, stateData, map[string]string{
		"fzf": "/usr/bin/fzf",
		"jq":  "/usr/bin/jq",
	})

	assertPresence(t, snapshot.Tools["fzf"], true, OwnershipManaged)
	assertPresence(t, snapshot.Tools["jq"], true, OwnershipExternal)
	assertPresence(t, snapshot.Tools["bat"], false, OwnershipManaged)
}

func TestReconcileMissingStateStillBuildsSnapshot(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	snapshot := Reconcile(registry, state.State{}, map[string]string{
		"fzf": "/usr/bin/fzf",
	})

	assertPresence(t, snapshot.Tools["fzf"], true, OwnershipExternal)
	assertPresence(t, snapshot.Tools["bat"], false, OwnershipMissing)
}

func assertPresence(t *testing.T, got ToolPresence, wantInstalled bool, wantOwnership string) {
	t.Helper()

	if got.Installed != wantInstalled {
		t.Fatalf("ToolPresence.Installed = %t, want %t", got.Installed, wantInstalled)
	}

	if got.Ownership != wantOwnership {
		t.Fatalf("ToolPresence.Ownership = %q, want %q", got.Ownership, wantOwnership)
	}
}
