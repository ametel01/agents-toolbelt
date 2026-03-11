package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

func TestGenerateGolden(t *testing.T) {
	t.Parallel()

	tools := []catalog.Tool{
		{Bin: "gh", Category: "forge", SkillExpose: true},
		{Bin: "rg", Category: "search", SkillExpose: true},
		{Bin: "fd", Category: "filesystem", SkillExpose: true},
		{Bin: "jq", Category: "json", SkillExpose: true},
		{Bin: "yq", Category: "yaml", SkillExpose: true},
		{Bin: "direnv", Category: "env_management", SkillExpose: true},
		{Bin: "hidden", Category: "search", SkillExpose: false},
	}

	got := Generate(tools)
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "golden_skill.md"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	if got != string(wantBytes) {
		t.Fatalf("Generate() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, string(wantBytes))
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	paths := []string{
		filepath.Join(baseDir, ".claude", "skills", "cli-tools", "SKILL.md"),
		filepath.Join(baseDir, ".agents", "skills", "cli-tools", "SKILL.md"),
	}

	if err := Write("content", paths); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	for _, path := range paths {
		//nolint:gosec // The path is generated from the test temp directory.
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", path, err)
		}

		if string(data) != "content" {
			t.Fatalf("file %q = %q, want %q", path, string(data), "content")
		}
	}
}

func TestGenerateEmpty(t *testing.T) {
	t.Parallel()

	got := Generate(nil)
	if got == "" {
		t.Fatal("Generate(nil) returned an empty string")
	}
}
