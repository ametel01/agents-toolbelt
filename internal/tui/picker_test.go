package tui

import (
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewPickerModelPreselectsMustTools(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "rg", Name: "ripgrep", Tier: catalog.TierMust, Category: "search"},
		{ID: "shellcheck", Name: "shellcheck", Tier: catalog.TierShould, Category: "linting"},
	}, discovery.Snapshot{})

	if !model.selected["rg"] {
		t.Fatal("must-have tools should be preselected")
	}

	if model.selected["shellcheck"] {
		t.Fatal("should-have tools should start unselected")
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
