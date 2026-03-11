package tui

import (
	"strings"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewPickerModelStartsWithNothingSelected(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "rg", Name: "ripgrep", Tier: catalog.TierMust, Category: "search"},
		{ID: "shellcheck", Name: "shellcheck", Tier: catalog.TierShould, Category: "linting"},
	}, discovery.Snapshot{})

	if len(model.selected) != 0 {
		t.Fatalf("len(model.selected) = %d, want 0", len(model.selected))
	}
}

func TestSpaceTogglesCurrentTool(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "shellcheck", Name: "shellcheck", Tier: catalog.TierShould, Category: "linting"},
	}, discovery.Snapshot{})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(PickerModel)
	if !got.selected["shellcheck"] {
		t.Fatal("space should select the current tool")
	}
}

func TestSearchFiltersRows(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "rg", Name: "ripgrep", Tier: catalog.TierMust, Category: "search"},
		{ID: "shellcheck", Name: "shellcheck", Tier: catalog.TierShould, Category: "linting"},
	}, discovery.Snapshot{})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	searching := updated.(PickerModel)
	updated, _ = searching.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("lint")})
	filtered := updated.(PickerModel)

	if len(filtered.rows) != 1 || filtered.rows[0].tool.ID != "shellcheck" {
		t.Fatalf("filtered rows = %#v, want only shellcheck", filtered.rows)
	}
}

func TestNiceToolsStartCollapsed(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "rg", Name: "ripgrep", Tier: catalog.TierMust, Category: "search"},
		{ID: "sqlite3", Name: "sqlite3", Tier: catalog.TierNice, Category: "database"},
	}, discovery.Snapshot{})

	if len(model.rows) != 2 {
		t.Fatalf("len(model.rows) = %d, want 2 (tool + more row)", len(model.rows))
	}

	model.cursor = 1
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	expanded := updated.(PickerModel)

	if !expanded.expandedNice {
		t.Fatal("space on the more row should expand nice tools")
	}
}

func TestWindowBoundsKeepsCursorVisible(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "gh", Name: "gh", Tier: catalog.TierMust, Category: "forge"},
		{ID: "rg", Name: "ripgrep", Tier: catalog.TierMust, Category: "search"},
		{ID: "fd", Name: "fd", Tier: catalog.TierMust, Category: "filesystem"},
		{ID: "jq", Name: "jq", Tier: catalog.TierMust, Category: "json"},
		{ID: "yq", Name: "yq", Tier: catalog.TierMust, Category: "yaml"},
		{ID: "direnv", Name: "direnv", Tier: catalog.TierMust, Category: "env_management"},
		{ID: "just", Name: "just", Tier: catalog.TierMust, Category: "task_runner"},
		{ID: "uv", Name: "uv", Tier: catalog.TierMust, Category: "python_runtime"},
	}, discovery.Snapshot{})

	model.height = 8
	model.cursor = len(model.rows) - 1

	start, end := model.windowBounds()
	if start >= end {
		t.Fatalf("windowBounds() = (%d, %d), want non-empty range", start, end)
	}

	if model.cursor < start || model.cursor >= end {
		t.Fatalf("cursor %d not visible in range [%d, %d)", model.cursor, start, end)
	}
}

func TestQuitClearsSelections(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "rg", Name: "ripgrep", Tier: catalog.TierMust, Category: "search"},
	}, discovery.Snapshot{})

	// Select the tool first.
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	selected := updated.(PickerModel)
	if !selected.selected["rg"] {
		t.Fatal("space should select the current tool")
	}

	// Quit with q — selections should be cleared.
	updated, _ = selected.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	quit := updated.(PickerModel)
	if len(quit.SelectedTools()) != 0 {
		t.Fatalf("SelectedTools() after quit = %d, want 0", len(quit.SelectedTools()))
	}
}

func TestHumanCategoryUsesFriendlyLabels(t *testing.T) {
	t.Parallel()

	if got := humanCategory("forge"); got != "Source Control / Forge" {
		t.Fatalf("humanCategory(%q) = %q, want %q", "forge", got, "Source Control / Forge")
	}
}

func TestViewShowsDescriptionAndMergedCategoryLabel(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{
			ID:          "jq",
			Name:        "jq",
			Tier:        catalog.TierMust,
			Category:    "json",
			Description: "Query and rewrite JSON output.",
		},
		{
			ID:          "yq",
			Name:        "yq",
			Tier:        catalog.TierMust,
			Category:    "yaml",
			Description: "Edit YAML config from scripts.",
		},
	}, discovery.Snapshot{})

	view := model.View()
	if strings.Count(view, "Structured Data") != 1 {
		t.Fatalf("View() = %q, want one merged Structured Data heading", view)
	}

	if !strings.Contains(view, "Query and rewrite JSON output.") {
		t.Fatalf("View() = %q, want tool description", view)
	}
}
