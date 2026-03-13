package pkgmgr

import (
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

func TestResolveDependenciesChoosesBootstrappableManager(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tool, ok := registry.ByID("uv")
	if !ok {
		t.Fatal("registry.ByID(\"uv\") did not find a tool")
	}

	dependencies := ResolveDependencies([]catalog.Tool{tool}, []Manager{
		commandManager{name: "apt", available: true},
	})
	if len(dependencies) != 1 {
		t.Fatalf("len(ResolveDependencies()) = %d, want 1", len(dependencies))
	}

	dependency := dependencies[0]
	if dependency.Name != "pipx" {
		t.Fatalf("dependency.Name = %q, want %q", dependency.Name, "pipx")
	}

	if dependency.Manager.Name() != "apt" {
		t.Fatalf("dependency.Manager.Name() = %q, want %q", dependency.Manager.Name(), "apt")
	}

	if dependency.Method.Package != "pipx" {
		t.Fatalf("dependency.Method.Package = %q, want %q", dependency.Method.Package, "pipx")
	}
}

func TestResolveDependenciesMergesToolsForSameDependency(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	yq, ok := registry.ByID("yq")
	if !ok {
		t.Fatal("registry.ByID(\"yq\") did not find a tool")
	}

	grpcurl, ok := registry.ByID("grpcurl")
	if !ok {
		t.Fatal("registry.ByID(\"grpcurl\") did not find a tool")
	}

	dependencies := ResolveDependencies([]catalog.Tool{yq, grpcurl}, []Manager{
		commandManager{name: "apt", available: true},
	})
	if len(dependencies) != 1 {
		t.Fatalf("len(ResolveDependencies()) = %d, want 1", len(dependencies))
	}

	dependency := dependencies[0]
	if dependency.Name != "go" {
		t.Fatalf("dependency.Name = %q, want %q", dependency.Name, "go")
	}

	if len(dependency.RequiredBy) != 2 {
		t.Fatalf("len(dependency.RequiredBy) = %d, want 2", len(dependency.RequiredBy))
	}
}

func TestResolveDependenciesSkipsAlreadyInstallableTools(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tool, ok := registry.ByID("uv")
	if !ok {
		t.Fatal("registry.ByID(\"uv\") did not find a tool")
	}

	dependencies := ResolveDependencies([]catalog.Tool{tool}, []Manager{
		commandManager{name: "apt", available: true},
		commandManager{name: "pipx", available: true},
	})
	if len(dependencies) != 0 {
		t.Fatalf("len(ResolveDependencies()) = %d, want 0", len(dependencies))
	}
}

func TestIsSecondaryManager(t *testing.T) {
	t.Parallel()

	if !IsSecondaryManager("cargo") {
		t.Fatal("IsSecondaryManager(\"cargo\") = false, want true")
	}

	if IsSecondaryManager("brew") {
		t.Fatal("IsSecondaryManager(\"brew\") = true, want false")
	}
}
