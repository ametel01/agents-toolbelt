package catalog

import "testing"

func TestLoadRegistry(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tools := registry.Tools()
	if len(tools) != 22 {
		t.Fatalf("len(registry.Tools()) = %d, want 22", len(tools))
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

	tool, ok := registry.ByID("fzf")
	if !ok {
		t.Fatal("registry.ByID(\"fzf\") did not find a tool")
	}

	if tool.Bin != "fzf" {
		t.Fatalf("tool.Bin = %q, want %q", tool.Bin, "fzf")
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
	tools := registry.ByCategory("navigation")

	if len(tools) == 0 {
		t.Fatal("registry.ByCategory(\"navigation\") returned no tools")
	}

	for _, tool := range tools {
		if tool.Category != "navigation" {
			t.Fatalf("tool %q has category %q, want %q", tool.ID, tool.Category, "navigation")
		}
	}
}

func TestForPlatform(t *testing.T) {
	t.Parallel()

	registry := mustLoadRegistry(t)

	for _, platform := range []string{"linux", "macos"} {
		tools := registry.ForPlatform(platform)
		if len(tools) != 22 {
			t.Fatalf("len(registry.ForPlatform(%q)) = %d, want 22", platform, len(tools))
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
