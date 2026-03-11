package main

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestUpdateCommandRunsSelfUpdate(t *testing.T) {
	t.Parallel()

	var called bool
	cmd := newUpdateCmdWithRunners(func(_ context.Context, _, _ io.Writer) error {
		called = true

		return nil
	}, runToolUpdate)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}

	if !called {
		t.Fatal("self-update runner was not called")
	}
}

func TestUpdateToolsCommandRunsToolUpdates(t *testing.T) {
	t.Parallel()

	var gotToolID string
	cmd := newUpdateCmdWithRunners(runSelfUpdate, func(_ context.Context, _, _ io.Writer, toolID string) error {
		gotToolID = toolID

		return nil
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"tools", "rg"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}

	if gotToolID != "rg" {
		t.Fatalf("toolID = %q, want %q", gotToolID, "rg")
	}
}
