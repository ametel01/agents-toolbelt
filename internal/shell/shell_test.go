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
		{ID: "direnv", Name: "direnv", ShellHook: "required"},
		{ID: "jq", Name: "jq", ShellHook: "none"},
	})

	if len(suggestions) != 1 {
		t.Fatalf("len(Suggestions()) = %d, want 1", len(suggestions))
	}

	if suggestions[0].InitLine != "eval \"$(direnv hook zsh)\"" {
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

func TestApplySkipsCommentedOutHook(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/bash")

	rcFile := filepath.Join(homeDir, ".bashrc")
	initLine := `eval "$(direnv hook bash)"`

	// Write a commented-out version of the init line.
	//nolint:gosec // Test fixture written to a temp directory.
	if err := os.WriteFile(rcFile, []byte("# "+initLine+"\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	suggestions := []Suggestion{{
		ToolID:   "direnv",
		ToolName: "direnv",
		Shell:    "bash",
		RCFile:   rcFile,
		InitLine: initLine,
		Required: true,
	}}

	var st state.State
	if err := ApplyConfirmedSuggestions(suggestions, &st); err != nil {
		t.Fatalf("ApplyConfirmedSuggestions() error = %v", err)
	}

	//nolint:gosec // The rc file path is derived from the test temp HOME.
	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", rcFile, err)
	}

	// The uncommented init line should have been appended.
	if strings.Count(string(data), initLine) != 2 { // one commented, one real
		t.Fatalf("rc file contents = %q, want both commented and uncommented init lines", string(data))
	}

	receipt, ok := st.Tool("direnv")
	if !ok {
		t.Fatal("state.Tool(\"direnv\") did not find a receipt")
	}

	if receipt.ShellHookStatus != shellHookApplied {
		t.Fatalf("receipt.ShellHookStatus = %q, want %q", receipt.ShellHookStatus, shellHookApplied)
	}
}

func TestContainsUncommentedLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		initLine string
		want     bool
	}{
		{
			name:     "exact match",
			content:  "eval \"$(direnv hook bash)\"\n",
			initLine: "eval \"$(direnv hook bash)\"",
			want:     true,
		},
		{
			name:     "commented out",
			content:  "# eval \"$(direnv hook bash)\"\n",
			initLine: "eval \"$(direnv hook bash)\"",
			want:     false,
		},
		{
			name:     "with leading spaces",
			content:  "  eval \"$(direnv hook bash)\"\n",
			initLine: "eval \"$(direnv hook bash)\"",
			want:     true,
		},
		{
			name:     "empty file",
			content:  "",
			initLine: "eval \"$(direnv hook bash)\"",
			want:     false,
		},
		{
			name:     "substring only",
			content:  "something eval \"$(direnv hook bash)\" extra\n",
			initLine: "eval \"$(direnv hook bash)\"",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := containsUncommentedLine(tt.content, tt.initLine); got != tt.want {
				t.Fatalf("containsUncommentedLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkDeclinedSuggestions(t *testing.T) {
	t.Parallel()

	var st state.State
	MarkDeclinedSuggestions([]Suggestion{{ToolID: "direnv"}}, &st)

	receipt, ok := st.Tool("direnv")
	if !ok {
		t.Fatal("state.Tool(\"direnv\") did not find a receipt")
	}

	if receipt.ShellHookStatus != shellHookDeclined {
		t.Fatalf("receipt.ShellHookStatus = %q, want %q", receipt.ShellHookStatus, shellHookDeclined)
	}
}
