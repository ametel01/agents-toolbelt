package catalog

import "testing"

func TestLoadRegistry(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tools := registry.Tools()
	if len(tools) == 0 {
		t.Fatal("registry.Tools() returned no tools")
	}

	bins := make(map[string]struct{}, len(tools))

	for _, tool := range tools {
		if tool.Bin == "" {
			t.Fatalf("tool %q has an empty bin", tool.ID)
		}

		if _, exists := bins[tool.Bin]; exists {
			t.Fatalf("duplicate bin %q", tool.Bin)
		}

		bins[tool.Bin] = struct{}{}

		if !isValidTier(tool.Tier) {
			t.Fatalf("tool %q has invalid tier %q", tool.ID, tool.Tier)
		}

		if len(tool.InstallMethods) == 0 {
			t.Fatalf("tool %q has no install methods", tool.ID)
		}

		if len(tool.Verify.Command) == 0 {
			t.Fatalf("tool %q has no verify command", tool.ID)
		}
	}
}

func TestByID(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)

	tool, ok := registry.ByID("rg")
	if !ok {
		t.Fatal("registry.ByID(\"rg\") did not find a tool")
	}

	if tool.Bin != "rg" {
		t.Fatalf("tool.Bin = %q, want %q", tool.Bin, "rg")
	}
}

func TestByTier(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tools := registry.ByTier(TierMust)

	if len(tools) == 0 {
		t.Fatal("registry.ByTier(TierMust) returned no tools")
	}

	for _, tool := range tools {
		if tool.Tier != TierMust {
			t.Fatalf("tool %q has tier %q, want %q", tool.ID, tool.Tier, TierMust)
		}
	}
}

func TestByCategory(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	tools := registry.ByCategory("search")

	if len(tools) == 0 {
		t.Fatal("registry.ByCategory(\"search\") returned no tools")
	}

	for _, tool := range tools {
		if tool.Category != "search" {
			t.Fatalf("tool %q has category %q, want %q", tool.ID, tool.Category, "search")
		}
	}
}

func TestCategoryLabelsComplete(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)

	for _, tool := range registry.Tools() {
		if _, ok := CategoryLabels[tool.Category]; !ok {
			t.Errorf("tool %q uses unmapped category %q", tool.ID, tool.Category)
		}
	}
}

func TestForPlatform(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)
	toolCount := len(registry.Tools())

	for _, platform := range []string{"linux", "macos"} {
		tools := registry.ForPlatform(platform)
		if len(tools) != toolCount {
			t.Fatalf("len(registry.ForPlatform(%q)) = %d, want %d", platform, len(tools), toolCount)
		}
	}
}

func mustLoadRegistry(t *testing.T) Registry {
	t.Helper()

	registry, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	return registry
}
