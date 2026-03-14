package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
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
