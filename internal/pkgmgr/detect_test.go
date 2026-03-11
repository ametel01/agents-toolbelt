package pkgmgr

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

func TestDetectManagers(t *testing.T) {
	tempDir := t.TempDir()
	createExecutable(t, filepath.Join(tempDir, "brew"))
	createExecutable(t, filepath.Join(tempDir, "go"))
	createExecutable(t, filepath.Join(tempDir, "cargo"))
	t.Setenv("PATH", tempDir)

	managers := DetectManagers()
	want := []string{"brew", "go", "cargo"}
	if len(managers) != len(want) {
		t.Fatalf("len(DetectManagers()) = %d, want %d", len(managers), len(want))
	}

	for index, manager := range managers {
		if manager.Name() != want[index] {
			t.Fatalf("manager[%d].Name() = %q, want %q", index, manager.Name(), want[index])
		}
	}
}

func TestSelectBest(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tool, ok := registry.ByID("rg")
	if !ok {
		t.Fatal("registry.ByID(\"rg\") did not find a tool")
	}

	available := []Manager{
		commandManager{name: "cargo", available: true},
		commandManager{name: "brew", available: true},
	}

	method, manager, err := SelectBest(tool, available)
	if err != nil {
		t.Fatalf("SelectBest() error = %v", err)
	}

	if manager.Name() != "brew" {
		t.Fatalf("manager.Name() = %q, want %q", manager.Name(), "brew")
	}

	if method.Manager != "brew" {
		t.Fatalf("method.Manager = %q, want %q", method.Manager, "brew")
	}
}

func TestSelectBestNoManagers(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tool, ok := registry.ByID("rg")
	if !ok {
		t.Fatal("registry.ByID(\"rg\") did not find a tool")
	}

	_, _, err = SelectBest(tool, nil)
	if err == nil {
		t.Fatal("SelectBest() error = nil, want ErrNoManagersDetected")
	}

	if !errors.Is(err, ErrNoManagersDetected) {
		t.Fatalf("SelectBest() error = %v, want %v", err, ErrNoManagersDetected)
	}
}

func TestSelectBestNoMatchingMethod(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tool, ok := registry.ByID("rg")
	if !ok {
		t.Fatal("registry.ByID(\"rg\") did not find a tool")
	}

	available := []Manager{
		commandManager{name: "pipx", available: true},
	}

	_, _, err = SelectBest(tool, available)
	if err == nil {
		t.Fatal("SelectBest() error = nil, want ErrNoMatchingMethod")
	}

	if !errors.Is(err, ErrNoMatchingMethod) {
		t.Fatalf("SelectBest() error = %v, want %v", err, ErrNoMatchingMethod)
	}
}

func createExecutable(t *testing.T, path string) {
	t.Helper()

	target, err := exec.LookPath("true")
	if err != nil {
		t.Fatalf("exec.LookPath(%q) error = %v", "true", err)
	}

	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("os.Symlink(%q, %q) error = %v", target, path, err)
	}
}
