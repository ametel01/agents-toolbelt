package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	original := State{
		Version:   1,
		LastRunAt: time.Unix(1_700_000_000, 0).UTC(),
		Tools: map[string]ToolState{
			"fzf": {
				ToolID:           "fzf",
				Bin:              "fzf",
				Ownership:        OwnershipManaged,
				InstallManager:   "brew",
				InstallPackage:   "fzf",
				InstallCommand:   []string{"brew", "install", "fzf"},
				UpdateCommand:    []string{"brew", "upgrade", "fzf"},
				UninstallCommand: []string{"brew", "uninstall", "fzf"},
				InstalledAt:      time.Unix(1_700_000_001, 0).UTC(),
				LastVerifyAt:     time.Unix(1_700_000_002, 0).UTC(),
				LastVerifyOK:     false,
				LastVerifyError:  "verify failed",
			},
		},
	}

	if err := saveToPath(path, original); err != nil {
		t.Fatalf("saveToPath() error = %v", err)
	}

	loaded, err := loadFromPath(path)
	if err != nil {
		t.Fatalf("loadFromPath() error = %v", err)
	}

	// LastRunAt is updated to the current time on save, so compare it separately.
	if loaded.LastRunAt.Before(original.LastRunAt) {
		t.Fatalf("loaded.LastRunAt = %v, want >= %v", loaded.LastRunAt, original.LastRunAt)
	}

	// Compare the remaining fields with LastRunAt normalized.
	loaded.LastRunAt = original.LastRunAt
	if !reflect.DeepEqual(loaded, original) {
		t.Fatalf("loadFromPath() = %#v, want %#v", loaded, original)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.json")
	loaded, err := loadFromPath(path)
	if err != nil {
		t.Fatalf("loadFromPath() error = %v", err)
	}

	if loaded.Version != 1 {
		t.Fatalf("loaded.Version = %d, want 1", loaded.Version)
	}

	if len(loaded.Tools) != 0 {
		t.Fatalf("len(loaded.Tools) = %d, want 0", len(loaded.Tools))
	}
}

func TestLoadCorruptJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := saveBytes(path, []byte("{not-json")); err != nil {
		t.Fatalf("saveBytes() error = %v", err)
	}

	loaded, err := loadFromPath(path)
	if err == nil {
		t.Fatal("loadFromPath() error = nil, want ErrCorruptState")
	}

	if !errors.Is(err, ErrCorruptState) {
		t.Fatalf("loadFromPath() error = %v, want %v", err, ErrCorruptState)
	}

	if len(loaded.Tools) != 0 {
		t.Fatalf("len(loaded.Tools) = %d, want 0", len(loaded.Tools))
	}
}

func TestAddReceiptSetsManagedOwnership(t *testing.T) {
	t.Parallel()

	var st State
	err := st.AddReceipt(ToolState{
		ToolID:           "fzf",
		Bin:              "fzf",
		InstallManager:   "brew",
		InstallPackage:   "fzf",
		InstallCommand:   []string{"brew", "install", "fzf"},
		UpdateCommand:    []string{"brew", "upgrade", "fzf"},
		UninstallCommand: []string{"brew", "uninstall", "fzf"},
	})
	if err != nil {
		t.Fatalf("AddReceipt() error = %v", err)
	}

	receipt, ok := st.Tool("fzf")
	if !ok {
		t.Fatal("State.Tool() did not find fzf")
	}

	if receipt.Ownership != OwnershipManaged {
		t.Fatalf("receipt.Ownership = %q, want %q", receipt.Ownership, OwnershipManaged)
	}

	if !reflect.DeepEqual(receipt.InstallCommand, []string{"brew", "install", "fzf"}) {
		t.Fatalf("receipt.InstallCommand = %#v, want %#v", receipt.InstallCommand, []string{"brew", "install", "fzf"})
	}
}

func TestMarkExternalSetsExternalOwnership(t *testing.T) {
	t.Parallel()

	var st State
	if err := st.MarkExternal("fzf", "fzf", "/usr/bin/fzf"); err != nil {
		t.Fatalf("MarkExternal() error = %v", err)
	}

	receipt, ok := st.Tool("fzf")
	if !ok {
		t.Fatal("State.Tool() did not find fzf")
	}

	if receipt.Ownership != OwnershipExternal {
		t.Fatalf("receipt.Ownership = %q, want %q", receipt.Ownership, OwnershipExternal)
	}
}

func TestIsATBManaged(t *testing.T) {
	t.Parallel()

	st := State{
		Version: 1,
		Tools: map[string]ToolState{
			"managed": {
				ToolID:    "managed",
				Ownership: OwnershipManaged,
			},
			"external": {
				ToolID:    "external",
				Ownership: OwnershipExternal,
			},
		},
	}

	if !st.IsATBManaged("managed") {
		t.Fatal("State.IsATBManaged(\"managed\") = false, want true")
	}

	if st.IsATBManaged("external") {
		t.Fatal("State.IsATBManaged(\"external\") = true, want false")
	}
}

func saveBytes(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}

	return nil
}
