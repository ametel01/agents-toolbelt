package verify

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/execx"
)

func TestCheckBinaryNotFound(t *testing.T) {
	t.Parallel()

	tool := catalog.Tool{
		ID:  "fzf",
		Bin: "fzf",
		Verify: catalog.VerifySpec{
			Command:           []string{"fzf", "--version"},
			TimeoutSeconds:    1,
			ExpectedExitCodes: []int{0},
		},
	}

	result, err := check(context.Background(), tool, fakeExecutor{}, func(string) (string, error) {
		return "", exec.ErrNotFound
	})
	if err != nil {
		t.Fatalf("check() error = %v", err)
	}

	if result.Found {
		t.Fatal("result.Found = true, want false")
	}
}

func TestCheckVerified(t *testing.T) {
	t.Parallel()

	tool := catalog.Tool{
		ID:  "fzf",
		Bin: "fzf",
		Verify: catalog.VerifySpec{
			Command:           []string{"fzf", "--version"},
			TimeoutSeconds:    1,
			ExpectedExitCodes: []int{0},
			VersionRegex:      "^([0-9]+\\.[0-9]+\\.[0-9]+)",
		},
	}

	result, err := check(context.Background(), tool, fakeExecutor{
		result: execx.Result{
			ExitCode: 0,
			Stdout:   "1.2.3\n",
		},
	}, foundOnPath)
	if err != nil {
		t.Fatalf("check() error = %v", err)
	}

	if !result.Verified {
		t.Fatal("result.Verified = false, want true")
	}

	if result.Version != "1.2.3" {
		t.Fatalf("result.Version = %q, want %q", result.Version, "1.2.3")
	}
}

func TestCheckTimeout(t *testing.T) {
	t.Parallel()

	tool := catalog.Tool{
		ID:  "fzf",
		Bin: "fzf",
		Verify: catalog.VerifySpec{
			Command:           []string{"fzf", "--version"},
			TimeoutSeconds:    1,
			ExpectedExitCodes: []int{0},
		},
	}

	result, err := check(context.Background(), tool, fakeExecutor{
		err: context.DeadlineExceeded,
	}, foundOnPath)
	if err != nil {
		t.Fatalf("check() error = %v", err)
	}

	if result.Verified {
		t.Fatal("result.Verified = true, want false")
	}

	if result.Error == "" {
		t.Fatal("result.Error = empty, want timeout message")
	}
}

func TestCheckUnexpectedExitCode(t *testing.T) {
	t.Parallel()

	tool := catalog.Tool{
		ID:  "fzf",
		Bin: "fzf",
		Verify: catalog.VerifySpec{
			Command:           []string{"fzf", "--version"},
			TimeoutSeconds:    1,
			ExpectedExitCodes: []int{0},
		},
	}

	result, err := check(context.Background(), tool, fakeExecutor{
		result: execx.Result{ExitCode: 2},
	}, foundOnPath)
	if err != nil {
		t.Fatalf("check() error = %v", err)
	}

	if result.Verified {
		t.Fatal("result.Verified = true, want false")
	}
}

func TestCheckVersionRegexMismatchDoesNotFail(t *testing.T) {
	t.Parallel()

	tool := catalog.Tool{
		ID:  "fzf",
		Bin: "fzf",
		Verify: catalog.VerifySpec{
			Command:           []string{"fzf", "--version"},
			TimeoutSeconds:    1,
			ExpectedExitCodes: []int{0},
			VersionRegex:      "version ([0-9]+)",
		},
	}

	result, err := check(context.Background(), tool, fakeExecutor{
		result: execx.Result{
			ExitCode: 0,
			Stdout:   "1.2.3\n",
		},
	}, foundOnPath)
	if err != nil {
		t.Fatalf("check() error = %v", err)
	}

	if !result.Verified {
		t.Fatal("result.Verified = false, want true")
	}

	if result.Version != "" {
		t.Fatalf("result.Version = %q, want empty", result.Version)
	}
}

func TestCheckInvalidRegex(t *testing.T) {
	t.Parallel()

	tool := catalog.Tool{
		ID:  "fzf",
		Bin: "fzf",
		Verify: catalog.VerifySpec{
			Command:           []string{"fzf", "--version"},
			TimeoutSeconds:    1,
			ExpectedExitCodes: []int{0},
			VersionRegex:      "(",
		},
	}

	_, err := check(context.Background(), tool, fakeExecutor{
		result: execx.Result{ExitCode: 0},
	}, foundOnPath)
	if err == nil {
		t.Fatal("check() error = nil, want invalid regex")
	}

	if !errors.Is(err, errInvalidVersionRegex) {
		t.Fatalf("check() error = %v, want %v", err, errInvalidVersionRegex)
	}
}

type fakeExecutor struct {
	err    error
	result execx.Result
}

func (f fakeExecutor) Run(_ context.Context, _ []string, _ time.Duration) (execx.Result, error) {
	return f.result, f.err
}

func foundOnPath(string) (string, error) {
	return "/usr/bin/tool", nil
}
