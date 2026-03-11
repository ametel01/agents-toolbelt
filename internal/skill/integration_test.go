package skill

import (
	"strings"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

func TestGenerateFromRealCatalogSubset(t *testing.T) {
	t.Parallel()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	tools := []catalog.Tool{
		mustTool(t, registry, "fzf"),
		mustTool(t, registry, "jq"),
		mustTool(t, registry, "direnv"),
	}

	content := Generate(tools)
	if !strings.Contains(content, "`fzf`") || !strings.Contains(content, "`jq`") || !strings.Contains(content, "`direnv`") {
		t.Fatalf("Generate() output did not contain expected tools:\n%s", content)
	}
}

func mustTool(t *testing.T, registry catalog.Registry, id string) catalog.Tool {
	t.Helper()

	tool, ok := registry.ByID(id)
	if !ok {
		t.Fatalf("registry.ByID(%q) did not find a tool", id)
	}

	return tool
}
