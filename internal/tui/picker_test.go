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
		{ID: "fzf", Name: "fzf", Tier: catalog.TierMust, Category: "navigation"},
		{ID: "starship", Name: "starship", Tier: catalog.TierShould, Category: "shell_prompt"},
	}, discovery.Snapshot{})

	if !model.selected["fzf"] {
		t.Fatal("must-have tools should be preselected")
	}

	if model.selected["starship"] {
		t.Fatal("should-have tools should start unselected")
	}
}

func TestSpaceTogglesCurrentTool(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "starship", Name: "starship", Tier: catalog.TierShould, Category: "shell_prompt"},
	}, discovery.Snapshot{})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(PickerModel)
	if !got.selected["starship"] {
		t.Fatal("space should select the current tool")
	}
}

func TestSearchFiltersRows(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "fzf", Name: "fzf", Tier: catalog.TierMust, Category: "navigation"},
		{ID: "starship", Name: "starship", Tier: catalog.TierShould, Category: "shell_prompt"},
	}, discovery.Snapshot{})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	searching := updated.(PickerModel)
	updated, _ = searching.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	filtered := updated.(PickerModel)

	if len(filtered.rows) != 1 || filtered.rows[0].tool.ID != "starship" {
		t.Fatalf("filtered rows = %#v, want only starship", filtered.rows)
	}
}

func TestNiceToolsStartCollapsed(t *testing.T) {
	t.Parallel()

	model := NewPickerModel([]catalog.Tool{
		{ID: "fzf", Name: "fzf", Tier: catalog.TierMust, Category: "navigation"},
		{ID: "k9s", Name: "k9s", Tier: catalog.TierNice, Category: "kubernetes_tui"},
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
