package execx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunSuccess(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), []string{"echo", "hello"}, time.Second)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Fatalf("result.ExitCode = %d, want 0", result.ExitCode)
	}

	if result.Stdout != "hello\n" {
		t.Fatalf("result.Stdout = %q, want %q", result.Stdout, "hello\n")
	}
}

func TestRunNonZeroExitCode(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), []string{"false"}, time.Second)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if result.ExitCode == 0 {
		t.Fatal("result.ExitCode = 0, want non-zero")
	}
}

func TestRunTimeout(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), []string{"sleep", "2"}, 10*time.Millisecond)
	if err == nil {
		t.Fatal("Run() error = nil, want timeout error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want deadline exceeded", err)
	}
}

func TestRunCommandNotFound(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), []string{"definitely-not-a-real-command"}, time.Second)
	if err == nil {
		t.Fatal("Run() error = nil, want command not found")
	}
}
