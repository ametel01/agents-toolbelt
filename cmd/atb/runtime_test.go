package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/skill"
	"github.com/ametel01/agents-toolbelt/internal/state"
	"github.com/ametel01/agents-toolbelt/internal/verify"
)

func TestRefreshVerifiedToolsIncludesExternalTools(t *testing.T) {
	registry := mustLoadRegistry(t)
	tempDir := t.TempDir()
	t.Setenv("PATH", tempDir)
	createExecutable(t, filepath.Join(tempDir, "rg"))
	createExecutable(t, filepath.Join(tempDir, "jq"))

	st := state.State{
		Version: 1,
		Tools: map[string]state.ToolState{
			"jq": {
				ToolID:         "jq",
				Bin:            "jq",
				Ownership:      state.OwnershipManaged,
				InstallManager: "brew",
			},
		},
	}

	verified, err := refreshVerifiedTools(context.Background(), registry, &st, fakeRuntimeVerifier{
		results: map[string]verify.VerifyResult{
			"rg": {ToolID: "rg", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
			"jq": {ToolID: "jq", Found: true, Verified: true, CheckedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatalf("refreshVerifiedTools() error = %v", err)
	}

	if !containsTool(verified, "rg") || !containsTool(verified, "jq") {
		t.Fatalf("verified tools = %#v, want rg and jq", verified)
	}

	rgReceipt, ok := st.Tool("rg")
	if !ok {
		t.Fatal("state.Tool(\"rg\") did not find an external receipt")
	}

	if rgReceipt.Ownership != state.OwnershipExternal {
		t.Fatalf("rg ownership = %q, want %q", rgReceipt.Ownership, state.OwnershipExternal)
	}

	if !rgReceipt.LastVerifyOK {
		t.Fatal("external rg receipt was not marked verified")
	}

	if rgReceipt.BinaryPath == "" {
		t.Fatal("external rg receipt did not record a binary path")
	}

	jqReceipt, ok := st.Tool("jq")
	if !ok {
		t.Fatal("state.Tool(\"jq\") did not find a managed receipt")
	}

	if jqReceipt.Ownership != state.OwnershipManaged {
		t.Fatalf("jq ownership = %q, want %q", jqReceipt.Ownership, state.OwnershipManaged)
	}
}

func TestRefreshVerifiedToolsClearsVersionForMissingBinary(t *testing.T) {
	registry := mustLoadRegistry(t)
	// Empty PATH so jq won't be found.
	t.Setenv("PATH", t.TempDir())

	st := state.State{
		Version: 1,
		Tools: map[string]state.ToolState{
			"jq": {
				ToolID:         "jq",
				Bin:            "jq",
				Ownership:      state.OwnershipManaged,
				InstallManager: "brew",
				BinaryPath:     "/usr/local/bin/jq",
				Version:        "1.7.1",
			},
		},
	}

	_, err := refreshVerifiedTools(context.Background(), registry, &st, fakeRuntimeVerifier{})
	if err != nil {
		t.Fatalf("refreshVerifiedTools() error = %v", err)
	}

	receipt, ok := st.Tool("jq")
	if !ok {
		t.Fatal("state.Tool(\"jq\") not found")
	}

	if receipt.BinaryPath != "" {
		t.Fatalf("receipt.BinaryPath = %q, want empty", receipt.BinaryPath)
	}

	if receipt.Version != "" {
		t.Fatalf("receipt.Version = %q, want empty", receipt.Version)
	}

	if receipt.LastVerifyOK {
		t.Fatal("receipt.LastVerifyOK = true, want false")
	}
}

func TestFinishInstallPersistsStateWhenTargetsCanceled(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	installCtx := &installContext{
		registry: mustLoadRegistry(t),
		stateData: state.State{
			Version: 1,
			Tools: map[string]state.ToolState{
				"jq": {
					ToolID:          "jq",
					Bin:             "jq",
					Ownership:       state.OwnershipManaged,
					InstallManager:  "brew",
					ShellHookStatus: "applied",
				},
			},
		},
	}

	var stdout bytes.Buffer

	// nil targets simulates the user canceling the target picker.
	if err := finishInstall(context.Background(), &stdout, installCtx, nil); err != nil {
		t.Fatalf("finishInstall() error = %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Skill generation skipped.")) {
		t.Fatalf("stdout = %q, want skip message", stdout.String())
	}

	statePath, err := state.DefaultPath()
	if err != nil {
		t.Fatalf("state.DefaultPath() error = %v", err)
	}

	//nolint:gosec // Test reads from a controlled temp directory.
	saved, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("state file not written: %v", err)
	}

	if !bytes.Contains(saved, []byte(`"jq"`)) {
		t.Fatalf("saved state missing jq receipt: %s", saved)
	}

	if !bytes.Contains(saved, []byte(`"applied"`)) {
		t.Fatalf("saved state missing shell hook status: %s", saved)
	}

	if !bytes.Contains(saved, []byte(`"none"`)) {
		t.Fatalf("saved state missing skill opt-out sentinel: %s", saved)
	}
}

func TestFinishInstallPersistsStateWithNormalTargets(t *testing.T) {
	registry := mustLoadRegistry(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	tempDir := t.TempDir()
	t.Setenv("PATH", tempDir)
	createExecutable(t, filepath.Join(tempDir, "jq"))

	installCtx := &installContext{
		registry: registry,
		stateData: state.State{
			Version: 1,
			Tools: map[string]state.ToolState{
				"jq": {
					ToolID:         "jq",
					Bin:            "jq",
					Ownership:      state.OwnershipManaged,
					InstallManager: "brew",
				},
			},
		},
	}

	var stdout bytes.Buffer

	targets := []skill.Target{{
		ID:      "test",
		Name:    "Test",
		RelPath: filepath.Join(".claude", "skills", "cli-tools", "SKILL.md"),
	}}

	if err := finishInstall(context.Background(), &stdout, installCtx, targets); err != nil {
		t.Fatalf("finishInstall() error = %v", err)
	}

	statePath, err := state.DefaultPath()
	if err != nil {
		t.Fatalf("state.DefaultPath() error = %v", err)
	}

	//nolint:gosec // Test reads from a controlled temp directory.
	if _, err := os.ReadFile(statePath); err != nil {
		t.Fatalf("state file not written on normal path: %v", err)
	}
}

func TestConfirmApplyFromInjectedInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"yes", "y\n", true},
		{"YES", "YES\n", true},
		{"no", "n\n", false},
		{"empty", "\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			got, err := confirmApply(strings.NewReader(tt.input), &stdout)
			if err != nil {
				t.Fatalf("confirmApply() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("confirmApply() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveStoredTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns all targets when state has no stored targets", func(t *testing.T) {
		t.Parallel()

		st := state.State{Version: 1}
		targets := resolveStoredTargets(st)

		if len(targets) != len(skill.AllTargets()) {
			t.Fatalf("len(targets) = %d, want %d", len(targets), len(skill.AllTargets()))
		}
	})

	t.Run("returns only stored targets", func(t *testing.T) {
		t.Parallel()

		st := state.State{Version: 1, SkillTargets: []string{"claude"}}
		targets := resolveStoredTargets(st)

		if len(targets) != 1 {
			t.Fatalf("len(targets) = %d, want 1", len(targets))
		}

		if targets[0].ID != "claude" {
			t.Fatalf("targets[0].ID = %q, want %q", targets[0].ID, "claude")
		}
	})

	t.Run("falls back to all when stored IDs are invalid", func(t *testing.T) {
		t.Parallel()

		st := state.State{Version: 1, SkillTargets: []string{"nonexistent"}}
		targets := resolveStoredTargets(st)

		if len(targets) != len(skill.AllTargets()) {
			t.Fatalf("len(targets) = %d, want %d (fallback to all)", len(targets), len(skill.AllTargets()))
		}
	})

	t.Run("returns nil when skill generation is opted out", func(t *testing.T) {
		t.Parallel()

		st := state.State{Version: 1, SkillTargets: []string{skillOptOut}}
		targets := resolveStoredTargets(st)

		if targets != nil {
			t.Fatalf("targets = %v, want nil for opt-out", targets)
		}
	})
}

func TestTargetIDs(t *testing.T) {
	t.Parallel()

	targets := skill.AllTargets()
	ids := targetIDs(targets)

	if len(ids) != len(targets) {
		t.Fatalf("len(ids) = %d, want %d", len(ids), len(targets))
	}

	for i, target := range targets {
		if ids[i] != target.ID {
			t.Fatalf("ids[%d] = %q, want %q", i, ids[i], target.ID)
		}
	}
}

func TestRunSelfUpdateRejectsDevelopmentBuilds(t *testing.T) {
	t.Parallel()

	previousVersion := version
	version = "dev"
	t.Cleanup(func() {
		version = previousVersion
	})

	var stdout bytes.Buffer
	err := runSelfUpdate(context.Background(), &stdout, io.Discard)
	if !errors.Is(err, errSelfUpdateDevBuild) {
		t.Fatalf("runSelfUpdate() error = %v, want %v", err, errSelfUpdateDevBuild)
	}
}

func TestInstallDependenciesStopsOnFailure(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	dependencies := []pkgmgr.DependencyPlanItem{
		{
			Name:    "cargo",
			Manager: failingManager{name: "apt"},
			Method:  catalog.InstallMethod{Manager: "apt", Command: []string{"echo", "install"}},
		},
	}

	err := installDependencies(context.Background(), &stdout, dependencies)
	if !errors.Is(err, errDependencyBootstrapFailed) {
		t.Fatalf("installDependencies() error = %v, want %v", err, errDependencyBootstrapFailed)
	}

	if !strings.Contains(err.Error(), "cargo") {
		t.Fatalf("error = %q, want mention of failed dependency name", err)
	}
}

func TestInstallDependenciesSucceedsWhenAllPass(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	dependencies := []pkgmgr.DependencyPlanItem{
		{
			Name:    "go",
			Manager: passingManager{name: "apt"},
			Method:  catalog.InstallMethod{Manager: "apt", Command: []string{"echo", "install"}},
		},
	}

	err := installDependencies(context.Background(), &stdout, dependencies)
	if err != nil {
		t.Fatalf("installDependencies() unexpected error = %v", err)
	}
}

var (
	errFakeInstall = errors.New("install failed")
	errBrokenStdin = errors.New("broken stdin")
	errWriteFailed = errors.New("write failed")
)

type failingManager struct{ name string }

func (m failingManager) Name() string    { return m.name }
func (m failingManager) Available() bool { return true }
func (m failingManager) Install(_ context.Context, _ catalog.InstallMethod) error {
	return errFakeInstall
}
func (m failingManager) Update(_ context.Context, _ catalog.InstallMethod) error    { return nil }
func (m failingManager) Uninstall(_ context.Context, _ catalog.InstallMethod) error { return nil }

type passingManager struct{ name string }

func (m passingManager) Name() string                                               { return m.name }
func (m passingManager) Available() bool                                            { return true }
func (m passingManager) Install(_ context.Context, _ catalog.InstallMethod) error   { return nil }
func (m passingManager) Update(_ context.Context, _ catalog.InstallMethod) error    { return nil }
func (m passingManager) Uninstall(_ context.Context, _ catalog.InstallMethod) error { return nil }

func mustLoadRegistry(t *testing.T) catalog.Registry {
	t.Helper()

	registry, err := catalog.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	return registry
}

func createExecutable(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}

	//nolint:gosec // Test helper needs an executable fixture on PATH.
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("os.Chmod(%q) error = %v", path, err)
	}
}

func containsTool(tools []catalog.Tool, id string) bool {
	for _, tool := range tools {
		if tool.ID == id {
			return true
		}
	}

	return false
}

type fakeRuntimeVerifier struct {
	results map[string]verify.VerifyResult
}

func (f fakeRuntimeVerifier) Check(_ context.Context, tool catalog.Tool) (verify.VerifyResult, error) {
	if result, ok := f.results[tool.ID]; ok {
		return result, nil
	}

	return verify.VerifyResult{ToolID: tool.ID, Found: true, Verified: true, CheckedAt: time.Now().UTC()}, nil
}

// --- Regression tests: early state persistence (issue #23) ---

func TestEarlySavePreservesReceiptsOnShellHookFailure(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	st := state.State{
		Version: 1,
		Tools: map[string]state.ToolState{
			"jq": {
				ToolID:         "jq",
				Bin:            "jq",
				Ownership:      state.OwnershipManaged,
				InstallManager: "brew",
			},
		},
	}

	// Simulate the early save that now runs after lifecycle execution.
	if err := state.Save(st); err != nil {
		t.Fatalf("early save: %v", err)
	}

	// Shell-hook prompting fails because the stdin reader is broken.
	direnvTool := catalog.Tool{
		ID:        "direnv",
		Bin:       "direnv",
		Name:      "direnv",
		ShellHook: "required",
	}

	err := applyShellWorkflow(
		iotest.ErrReader(errBrokenStdin),
		io.Discard,
		false,
		&st,
		[]catalog.Tool{direnvTool},
	)
	if err == nil {
		t.Fatal("expected applyShellWorkflow to fail")
	}

	assertStateOnDiskContains(t, "jq")
}

func TestEarlySavePreservesReceiptsOnSkillOutputWriteFailure(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("PATH", t.TempDir())

	registry := mustLoadRegistry(t)
	st := state.State{
		Version: 1,
		Tools: map[string]state.ToolState{
			"jq": {
				ToolID:              "jq",
				Bin:                 "jq",
				Ownership:           state.OwnershipManaged,
				InstallManager:      "brew",
				LastUpdateAttemptAt: time.Now().UTC(),
			},
		},
	}

	// Simulate the early save that now runs after lifecycle execution.
	if err := state.Save(st); err != nil {
		t.Fatalf("early save: %v", err)
	}

	targets := []skill.Target{{
		ID:      "test",
		Name:    "Test",
		RelPath: filepath.Join(".test", "SKILL.md"),
	}}

	// persistVerifiedSkill fails on the first stdout write, before its own
	// internal state.Save call. The early save must protect the receipts.
	err := persistVerifiedSkill(
		context.Background(),
		registry,
		&st,
		fakeRuntimeVerifier{},
		failingWriter{},
		targets,
	)
	if err == nil {
		t.Fatal("expected persistVerifiedSkill to fail")
	}

	assertStateOnDiskContains(t, "jq")
}

func TestEarlySavePreservesRemovalsOnSkillOutputWriteFailure(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("PATH", t.TempDir())

	registry := mustLoadRegistry(t)

	// After uninstall the tool receipt has been removed from state.
	st := state.State{
		Version: 1,
		Tools:   map[string]state.ToolState{},
	}

	// Simulate the early save that now runs after lifecycle execution.
	if err := state.Save(st); err != nil {
		t.Fatalf("early save: %v", err)
	}

	targets := []skill.Target{{
		ID:      "test",
		Name:    "Test",
		RelPath: filepath.Join(".test", "SKILL.md"),
	}}

	err := persistVerifiedSkill(
		context.Background(),
		registry,
		&st,
		fakeRuntimeVerifier{},
		failingWriter{},
		targets,
	)
	if err == nil {
		t.Fatal("expected persistVerifiedSkill to fail")
	}

	// The saved state must reflect the uninstall: jq must not be present.
	statePath, pathErr := state.DefaultPath()
	if pathErr != nil {
		t.Fatalf("state.DefaultPath(): %v", pathErr)
	}

	//nolint:gosec // Test reads from a controlled temp directory.
	saved, readErr := os.ReadFile(statePath)
	if readErr != nil {
		t.Fatalf("state file not written: %v", readErr)
	}

	if bytes.Contains(saved, []byte(`"jq"`)) {
		t.Fatalf("saved state still contains jq after uninstall: %s", saved)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

func assertStateOnDiskContains(t *testing.T, toolID string) {
	t.Helper()

	statePath, err := state.DefaultPath()
	if err != nil {
		t.Fatalf("state.DefaultPath(): %v", err)
	}

	//nolint:gosec // Test reads from a controlled temp directory.
	saved, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("state file not written: %v", err)
	}

	if !bytes.Contains(saved, []byte(`"`+toolID+`"`)) {
		t.Fatalf("saved state missing %s receipt: %s", toolID, saved)
	}
}
