// Package pkgmgr detects and selects supported package managers.
package pkgmgr

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

var (
	// ErrNoManagersDetected indicates that no supported package manager was found on PATH.
	ErrNoManagersDetected = errors.New("no supported package managers detected")
	// ErrNoMatchingMethod indicates that a tool has no install method for the detected managers.
	ErrNoMatchingMethod = errors.New("no matching install method")
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
	return runCommand(ctx, method.Command)
}

func (m commandManager) Update(ctx context.Context, method catalog.InstallMethod) error {
	return runCommand(ctx, method.UpdateCommand)
}

func (m commandManager) Uninstall(ctx context.Context, method catalog.InstallMethod) error {
	return runCommand(ctx, method.UninstallCommand)
}

func newCommandManager(name string, lookPath lookPathFunc) Manager {
	_, err := lookPath(name)

	return commandManager{
		name:      name,
		available: err == nil,
	}
}

func runCommand(ctx context.Context, args []string) error {
	//nolint:gosec // Command arguments come from the embedded catalog, not from user input.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run %q: %w: %s", args[0], err, output)
	}

	return nil
}
