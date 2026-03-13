package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	tea "github.com/charmbracelet/bubbletea"
)

var testDependencies = []pkgmgr.DependencyPlanItem{
	{
		Name: "pipx",
		Method: catalog.InstallMethod{
			Package: "pipx",
		},
		Manager: fakeDependencyManager{name: "apt"},
		RequiredBy: []catalog.Tool{
			{Name: "uv"},
		},
	},
	{
		Name: "cargo",
		Method: catalog.InstallMethod{
			Package: "rust",
		},
		Manager: fakeDependencyManager{name: "brew"},
		RequiredBy: []catalog.Tool{
			{Name: "fd"},
			{Name: "just"},
		},
	},
}

func TestNewDependencyModelStartsAllSelected(t *testing.T) {
	t.Parallel()

	model := NewDependencyModel(testDependencies)
	if len(model.SelectedDependencies()) != 2 {
		t.Fatalf("SelectedDependencies() = %d, want 2", len(model.SelectedDependencies()))
	}
}

func TestDependencySpaceToggles(t *testing.T) {
	t.Parallel()

	model := NewDependencyModel(testDependencies)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(DependencyModel)

	if got.selected["pipx"] {
		t.Fatal("space should deselect the current dependency")
	}

	if !got.selected["cargo"] {
		t.Fatal("cargo should remain selected")
	}
}

func TestDependencyNoneSelectionAllowed(t *testing.T) {
	t.Parallel()

	model := NewDependencyModel(testDependencies)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	got := updated.(DependencyModel)

	if len(got.SelectedDependencies()) != 0 {
		t.Fatalf("SelectedDependencies() = %d, want 0", len(got.SelectedDependencies()))
	}
}

func TestDependencyQuitCancels(t *testing.T) {
	t.Parallel()

	model := NewDependencyModel(testDependencies)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := updated.(DependencyModel)

	if got.SelectedDependencies() != nil {
		t.Fatal("SelectedDependencies() after quit = non-nil, want nil")
	}
}

func TestDependencyViewIncludesRequiredBy(t *testing.T) {
	t.Parallel()

	model := NewDependencyModel(testDependencies)
	view := model.View()

	if !strings.Contains(view, "Required by: uv") {
		t.Fatalf("View() = %q, want uv requirement", view)
	}

	if !strings.Contains(view, "cargo (rust package) via brew") {
		t.Fatalf("View() = %q, want package note for cargo", view)
	}
}

type fakeDependencyManager struct {
	name string
}

func (f fakeDependencyManager) Name() string {
	return f.name
}

func (f fakeDependencyManager) Available() bool {
	return true
}

func (f fakeDependencyManager) Install(_ context.Context, _ catalog.InstallMethod) error {
	return nil
}

func (f fakeDependencyManager) Update(_ context.Context, _ catalog.InstallMethod) error {
	return nil
}

func (f fakeDependencyManager) Uninstall(_ context.Context, _ catalog.InstallMethod) error {
	return nil
}
