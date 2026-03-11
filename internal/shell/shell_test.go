package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/state"
)

func TestDetectShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	if got := DetectShell(); got != "zsh" {
		t.Fatalf("DetectShell() = %q, want %q", got, "zsh")
	}

	t.Setenv("SHELL", "/usr/bin/fish")
	if got := DetectShell(); got != "fish" {
		t.Fatalf("DetectShell() = %q, want %q", got, "fish")
	}

	t.Setenv("SHELL", "/bin/bash")
	if got := DetectShell(); got != "bash" {
		t.Fatalf("DetectShell() = %q, want %q", got, "bash")
	}
}

func TestSuggestions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/zsh")

	suggestions := Suggestions([]catalog.Tool{
		{ID: "zoxide", Name: "zoxide", ShellHook: "required"},
		{ID: "starship", Name: "starship", ShellHook: "optional"},
		{ID: "jq", Name: "jq", ShellHook: "none"},
	})

	if len(suggestions) != 2 {
		t.Fatalf("len(Suggestions()) = %d, want 2", len(suggestions))
	}

	if suggestions[0].InitLine != "eval \"$(zoxide init zsh)\"" {
		t.Fatalf("suggestions[0].InitLine = %q", suggestions[0].InitLine)
	}
}

func TestApplyConfirmedSuggestionsIsIdempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/bash")

	rcFile := filepath.Join(os.Getenv("HOME"), ".bashrc")
	suggestions := []Suggestion{{
		ToolID:   "direnv",
		ToolName: "direnv",
		Shell:    "bash",
		RCFile:   rcFile,
		InitLine: "eval \"$(direnv hook bash)\"",
		Required: true,
	}}

	var st state.State
	if err := ApplyConfirmedSuggestions(suggestions, &st); err != nil {
		t.Fatalf("ApplyConfirmedSuggestions() error = %v", err)
	}

	if err := ApplyConfirmedSuggestions(suggestions, &st); err != nil {
		t.Fatalf("ApplyConfirmedSuggestions() second call error = %v", err)
	}

	//nolint:gosec // The rc file path is derived from the test temp HOME.
	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", rcFile, err)
	}

	if strings.Count(string(data), suggestions[0].InitLine) != 1 {
		t.Fatalf("rc file contents = %q, want one init line", string(data))
	}

	receipt, ok := st.Tool("direnv")
	if !ok {
		t.Fatal("state.Tool(\"direnv\") did not find a receipt")
	}

	if receipt.ShellHookStatus != shellHookApplied {
		t.Fatalf("receipt.ShellHookStatus = %q, want %q", receipt.ShellHookStatus, shellHookApplied)
	}
}

func TestMarkDeclinedSuggestions(t *testing.T) {
	t.Parallel()

	var st state.State
	MarkDeclinedSuggestions([]Suggestion{{ToolID: "zoxide"}}, &st)

	receipt, ok := st.Tool("zoxide")
	if !ok {
		t.Fatal("state.Tool(\"zoxide\") did not find a receipt")
	}

	if receipt.ShellHookStatus != shellHookDeclined {
		t.Fatalf("receipt.ShellHookStatus = %q, want %q", receipt.ShellHookStatus, shellHookDeclined)
	}
}
