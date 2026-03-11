// Package shell renders and applies shell-hook suggestions for installed tools.
package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/state"
)

const (
	shellHookApplied   = "applied"
	shellHookDeclined  = "declined"
	shellHookSuggested = "suggested"
)

// Suggestion describes a shell initialization line for a tool.
type Suggestion struct {
	ToolID   string
	ToolName string
	Shell    string
	RCFile   string
	InitLine string
	Required bool
}

// DetectShell returns the normalized current user shell.
func DetectShell() string {
	shellPath := os.Getenv("SHELL")
	switch filepath.Base(shellPath) {
	case "zsh":
		return "zsh"
	case "fish":
		return "fish"
	default:
		return "bash"
	}
}

// Suggestions returns init-line suggestions for tools with shell hooks.
func Suggestions(tools []catalog.Tool) []Suggestion {
	shellName := DetectShell()
	rcFile, err := rcFileForShell(shellName)
	if err != nil {
		return nil
	}

	suggestions := make([]Suggestion, 0, len(tools))
	for _, tool := range tools {
		if tool.ShellHook != "required" && tool.ShellHook != "optional" {
			continue
		}

		initLine, ok := initLineForTool(tool.ID, shellName)
		if !ok {
			continue
		}

		suggestions = append(suggestions, Suggestion{
			ToolID:   tool.ID,
			ToolName: tool.Name,
			Shell:    shellName,
			RCFile:   rcFile,
			InitLine: initLine,
			Required: tool.ShellHook == "required",
		})
	}

	return suggestions
}

// ApplyConfirmedSuggestions appends missing init lines and records applied state.
func ApplyConfirmedSuggestions(suggestions []Suggestion, st *state.State) error {
	for _, suggestion := range suggestions {
		if err := appendInitLine(suggestion.RCFile, suggestion.InitLine); err != nil {
			return err
		}

		recordHookStatus(st, suggestion.ToolID, shellHookApplied, time.Now().UTC())
	}

	return nil
}

// MarkDeclinedSuggestions records that the user declined shell-hook changes.
func MarkDeclinedSuggestions(suggestions []Suggestion, st *state.State) {
	for _, suggestion := range suggestions {
		recordHookStatus(st, suggestion.ToolID, shellHookDeclined, time.Now().UTC())
	}
}

// MarkSuggestedSuggestions records that shell-hook changes were offered to the user.
func MarkSuggestedSuggestions(suggestions []Suggestion, st *state.State) {
	for _, suggestion := range suggestions {
		recordHookStatus(st, suggestion.ToolID, shellHookSuggested, time.Now().UTC())
	}
}

func appendInitLine(path, initLine string) error {
	//nolint:gosec // The rc file path is derived from the detected shell and user home directory.
	content, err := os.ReadFile(path)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		content = nil
	default:
		return fmt.Errorf("read rc file %s: %w", path, err)
	}

	if strings.Contains(string(content), initLine) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create rc file directory for %s: %w", path, err)
	}

	var builder strings.Builder
	builder.Write(content)
	if len(content) > 0 && !strings.HasSuffix(builder.String(), "\n") {
		builder.WriteString("\n")
	}

	builder.WriteString(initLine)
	builder.WriteString("\n")

	if err := os.WriteFile(path, []byte(builder.String()), 0o600); err != nil {
		return fmt.Errorf("write rc file %s: %w", path, err)
	}

	return nil
}

func initLineForTool(toolID, shellName string) (string, bool) {
	switch toolID {
	case "zoxide":
		return shellInit(shellName, "zoxide", "init"), true
	case "atuin":
		return shellInit(shellName, "atuin", "init"), true
	case "direnv":
		return shellInit(shellName, "direnv", "hook"), true
	case "starship":
		return shellInit(shellName, "starship", "init"), true
	default:
		return "", false
	}
}

func shellInit(shellName, tool, action string) string {
	if shellName == "fish" {
		return fmt.Sprintf("%s %s %s | source", tool, action, shellName)
	}

	return fmt.Sprintf("eval \"$(%s %s %s)\"", tool, action, shellName)
}

func rcFileForShell(shellName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}

	switch shellName {
	case "zsh":
		return filepath.Join(homeDir, ".zshrc"), nil
	case "fish":
		return filepath.Join(homeDir, ".config", "fish", "config.fish"), nil
	default:
		return filepath.Join(homeDir, ".bashrc"), nil
	}
}

func recordHookStatus(st *state.State, toolID, status string, at time.Time) {
	if st.Tools == nil {
		st.Tools = make(map[string]state.ToolState)
	}

	receipt := st.Tools[toolID]
	receipt.ToolID = toolID
	receipt.ShellHookStatus = status
	if status == shellHookSuggested {
		receipt.ShellHookSuggestedAt = at
	}

	if status == shellHookApplied {
		receipt.ShellHookAppliedAt = at
	}

	st.Tools[toolID] = receipt
}
