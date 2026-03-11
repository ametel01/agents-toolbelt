package pkgmgr

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCommandExpandsEnvironmentVariables(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	target := filepath.Join(homeDir, "expanded")

	if err := runCommand(context.Background(), []string{"touch", "$HOME/expanded"}, 5); err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", target, err)
	}
}

func TestRunCommandHonorsTimeout(t *testing.T) {
	t.Parallel()

	err := runCommand(context.Background(), []string{"sleep", "2"}, 1)
	if err == nil {
		t.Fatal("runCommand() error = nil, want timeout")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runCommand() error = %v, want deadline exceeded", err)
	}
}
