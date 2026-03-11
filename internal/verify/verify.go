// Package verify checks whether a catalog tool is installed and functional.
package verify

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/execx"
)

var errInvalidVersionRegex = errors.New("invalid version regex")

// Executor runs a command and returns its captured result.
type Executor interface {
	Run(ctx context.Context, args []string, timeout time.Duration) (execx.Result, error)
}

// LookPather abstracts executable lookup for tests.
type LookPather func(string) (string, error)

// VerifyResult captures the outcome of a tool verification attempt.
//
//nolint:revive // The implementation plan specifies the exported name VerifyResult.
type VerifyResult struct {
	ToolID    string
	Found     bool
	Verified  bool
	Version   string
	Error     string
	CheckedAt time.Time
}

// Check verifies that a tool is present and that its verification command succeeds.
func Check(ctx context.Context, tool catalog.Tool, executor Executor) (VerifyResult, error) {
	return check(ctx, tool, executor, exec.LookPath)
}

func check(ctx context.Context, tool catalog.Tool, executor Executor, lookPath LookPather) (VerifyResult, error) {
	result := VerifyResult{
		ToolID:    tool.ID,
		CheckedAt: time.Now(),
	}

	if _, err := lookPath(tool.Bin); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return result, nil
		}

		return result, fmt.Errorf("look up %q on PATH: %w", tool.Bin, err)
	}

	result.Found = true

	execResult, runErr := executor.Run(ctx, tool.Verify.Command, time.Duration(tool.Verify.TimeoutSeconds)*time.Second)
	if runErr != nil {
		result.Error = runErr.Error()

		//nolint:nilerr // Verification failures are reported in VerifyResult so callers can continue.
		return result, nil
	}

	if !slices.Contains(tool.Verify.ExpectedExitCodes, execResult.ExitCode) {
		result.Error = fmt.Sprintf("unexpected exit code: %d", execResult.ExitCode)

		return result, nil
	}

	result.Verified = true

	if tool.Verify.VersionRegex == "" {
		return result, nil
	}

	versionExpr, compileErr := regexp.Compile(tool.Verify.VersionRegex)
	if compileErr != nil {
		return result, errors.Join(
			errInvalidVersionRegex,
			fmt.Errorf("tool %s: %w", tool.ID, compileErr),
		)
	}

	versionMatch := versionExpr.FindStringSubmatch(execResult.Stdout + execResult.Stderr)
	if len(versionMatch) > 1 {
		result.Version = versionMatch[1]
	}

	return result, nil
}

// ExecExecutor executes verification commands through execx.
type ExecExecutor struct{}

// Run delegates execution to execx.Run.
func (ExecExecutor) Run(ctx context.Context, args []string, timeout time.Duration) (execx.Result, error) {
	result, err := execx.Run(ctx, args, timeout)
	if err != nil {
		return result, fmt.Errorf("execute verify command: %w", err)
	}

	return result, nil
}
