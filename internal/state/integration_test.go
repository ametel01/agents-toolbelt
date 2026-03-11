package state

import (
	"reflect"
	"testing"
)

func TestAddReceiptIsIdempotent(t *testing.T) {
	t.Parallel()

	var st State
	receipt := ToolState{
		ToolID:           "fzf",
		Bin:              "fzf",
		Ownership:        OwnershipManaged,
		InstallManager:   "brew",
		InstallPackage:   "fzf",
		InstallCommand:   []string{"brew", "install", "fzf"},
		UpdateCommand:    []string{"brew", "upgrade", "fzf"},
		UninstallCommand: []string{"brew", "uninstall", "fzf"},
	}

	if err := st.AddReceipt(receipt); err != nil {
		t.Fatalf("AddReceipt() error = %v", err)
	}

	first := st.Tools["fzf"]
	if err := st.AddReceipt(receipt); err != nil {
		t.Fatalf("AddReceipt() second call error = %v", err)
	}

	second := st.Tools["fzf"]
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("state after repeated AddReceipt() changed\nfirst: %#v\nsecond: %#v", first, second)
	}
}
