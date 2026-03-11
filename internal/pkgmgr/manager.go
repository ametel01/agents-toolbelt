// Package pkgmgr detects and selects supported package managers.
package pkgmgr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

var (
	// ErrNoManagersDetected indicates that no supported package manager was found on PATH.
	ErrNoManagersDetected = errors.New("no supported package managers detected")
	// ErrNoMatchingMethod indicates that a tool has no install method for the detected managers.
	ErrNoMatchingMethod = errors.New("no matching install method")
	errEmptyCommandArgs = errors.New("command arguments are required")
)

// Manager executes tool lifecycle actions through a specific package manager.
type Manager interface {
	Name() string
	Available() bool
	Install(ctx context.Context, method catalog.InstallMethod) error
	Update(ctx context.Context, method catalog.InstallMethod) error
	Uninstall(ctx context.Context, method catalog.InstallMethod) error
}

type commandManager struct {
	name      string
	available bool
}

func (m commandManager) Name() string {
	return m.name
}

func (m commandManager) Available() bool {
	return m.available
}

func (m commandManager) Install(ctx context.Context, method catalog.InstallMethod) error {
	return runCommand(ctx, method.Command, method.TimeoutSeconds)
}

func (m commandManager) Update(ctx context.Context, method catalog.InstallMethod) error {
	return runCommand(ctx, method.UpdateCommand, method.TimeoutSeconds)
}

func (m commandManager) Uninstall(ctx context.Context, method catalog.InstallMethod) error {
	return runCommand(ctx, method.UninstallCommand, method.TimeoutSeconds)
}

func newCommandManager(name string, lookPath lookPathFunc) Manager {
	_, err := lookPath(name)

	return commandManager{
		name:      name,
		available: err == nil,
	}
}

func runCommand(ctx context.Context, args []string, timeoutSeconds int) error {
	if len(args) == 0 {
		return errEmptyCommandArgs
	}

	commandArgs := expandArgs(args)
	runCtx := ctx
	cancel := func() {}
	if timeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	}
	defer cancel()

	//nolint:gosec // Command arguments come from the embedded catalog, not from user input.
	cmd := exec.CommandContext(runCtx, commandArgs[0], commandArgs[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("run %q: %w", commandArgs[0], context.DeadlineExceeded)
		}

		return fmt.Errorf("run %q: %w: %s", commandArgs[0], err, output)
	}

	return nil
}

func expandArgs(args []string) []string {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		expanded = append(expanded, os.ExpandEnv(arg))
	}

	return expanded
}
