// Package execx executes external commands safely without shell expansion.
package execx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

var (
	errCommandKilled = errors.New("command killed by signal")
	errEmptyArgs     = errors.New("command arguments are required")
)

// Result captures the observable outcome of a command execution.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// Run executes a command with a timeout, capturing stdout, stderr, and exit code.
func Run(ctx context.Context, args []string, timeout time.Duration) (Result, error) {
	if len(args) == 0 {
		return Result{}, errEmptyArgs
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	//nolint:gosec // Command arguments are structured inputs supplied by internal call sites.
	cmd := exec.CommandContext(runCtx, args[0], args[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(startedAt),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if err == nil {
		return result, nil
	}

	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return result, fmt.Errorf("command timed out: %w", context.DeadlineExceeded)
	}

	if errors.Is(err, exec.ErrNotFound) {
		return result, fmt.Errorf("command not found: %w", err)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok && waitStatus.Signaled() {
			return result, fmt.Errorf("%w: %s", errCommandKilled, waitStatus.Signal())
		}

		return result, nil
	}

	return result, fmt.Errorf("run command %q: %w", args[0], err)
}
