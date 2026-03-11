package tui

import (
	"strings"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/skill"
	tea "github.com/charmbracelet/bubbletea"
)

var testTargets = []skill.Target{
	{ID: "claude", Name: "Claude Code", RelPath: ".claude/skills/cli-tools/SKILL.md"},
	{ID: "codex", Name: "Codex", RelPath: ".agents/skills/cli-tools/SKILL.md"},
}

func TestNewTargetModelStartsAllSelected(t *testing.T) {
	t.Parallel()

	model := NewTargetModel(testTargets)
	if len(model.SelectedTargets()) != 2 {
		t.Fatalf("SelectedTargets() = %d, want 2", len(model.SelectedTargets()))
	}
}

func TestTargetSpaceToggles(t *testing.T) {
	t.Parallel()

	model := NewTargetModel(testTargets)

	// Deselect first target (Claude Code).
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(TargetModel)
	if got.selected["claude"] {
		t.Fatal("space should deselect the current target")
	}

	if !got.selected["codex"] {
		t.Fatal("codex should remain selected")
	}
}

func TestTargetSelectAll(t *testing.T) {
	t.Parallel()

	model := NewTargetModel(testTargets)

	// Deselect first, then select all.
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := updated.(TargetModel)

	if !got.selected["claude"] || !got.selected["codex"] {
		t.Fatal("'a' should select all targets")
	}
}

func TestTargetQuitCancels(t *testing.T) {
	t.Parallel()

	model := NewTargetModel(testTargets)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := updated.(TargetModel)

	if len(got.SelectedTargets()) != 0 {
		t.Fatalf("SelectedTargets() after quit = %d, want 0", len(got.SelectedTargets()))
	}
}

func TestTargetEnterBlockedWhenNoneSelected(t *testing.T) {
	t.Parallel()

	model := NewTargetModel(testTargets)

	// Deselect both.
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Enter should not quit.
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("enter should not quit when no targets are selected")
	}

	got := updated.(TargetModel)
	if got.cancelled {
		t.Fatal("model should not be cancelled")
	}
}

func TestTargetViewShowsBothTargets(t *testing.T) {
	t.Parallel()

	model := NewTargetModel(testTargets)
	view := model.View()

	if !strings.Contains(view, "Claude Code") {
		t.Fatal("view should contain Claude Code")
	}

	if !strings.Contains(view, "Codex") {
		t.Fatal("view should contain Codex")
	}
}
