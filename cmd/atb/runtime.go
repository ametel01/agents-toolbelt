package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/plan"
	"github.com/ametel01/agents-toolbelt/internal/platform"
	"github.com/ametel01/agents-toolbelt/internal/shell"
	"github.com/ametel01/agents-toolbelt/internal/skill"
	"github.com/ametel01/agents-toolbelt/internal/state"
	"github.com/ametel01/agents-toolbelt/internal/tui"
	"github.com/ametel01/agents-toolbelt/internal/verify"
)

var errNoSupportedPackageManagers = errors.New("no supported package managers detected")

type liveVerifier struct{}

type toolVerifier interface {
	Check(context.Context, catalog.Tool) (verify.VerifyResult, error)
}

func (liveVerifier) Check(ctx context.Context, tool catalog.Tool) (verify.VerifyResult, error) {
	result, err := verify.Check(ctx, tool, verify.ExecExecutor{})
	if err != nil {
		return result, wrapError("verify tool", err)
	}

	return result, nil
}

func runInstall(ctx context.Context, stdout, stderr io.Writer, yes bool) error {
	registry, err := catalog.LoadRegistry()
	if err != nil {
		return wrapError("load catalog", err)
	}

	managers := pkgmgr.DetectManagers()
	if len(managers) == 0 {
		return errNoSupportedPackageManagers
	}

	stateData, err := loadState(stderr)
	if err != nil {
		return wrapError("load state", err)
	}

	snapshot, err := buildSnapshot(registry, stateData)
	if err != nil {
		return wrapError("build discovery snapshot", err)
	}

	selected, err := selectTools(registry, snapshot, yes)
	if err != nil {
		return wrapError("select tools", err)
	}

	if len(selected) == 0 {
		_, writeErr := fmt.Fprintln(stdout, "No tools selected.")

		return wrapError("write empty selection message", writeErr)
	}

	installPlan, err := plan.BuildInstallPlan(selected, snapshot, managers)
	if err != nil {
		return wrapError("build install plan", err)
	}

	summary, err := plan.ExecuteInstallPlan(ctx, installPlan, &stateData, liveVerifier{})
	if err != nil {
		return wrapError("execute install plan", err)
	}

	if err := applyShellWorkflow(stdout, yes, &stateData, selected); err != nil {
		return wrapError("apply shell workflow", err)
	}

	if err := persistVerifiedSkill(ctx, registry, &stateData, liveVerifier{}); err != nil {
		return err
	}

	renderSummary(stdout, "install", summary)

	return nil
}

func runStatus(stdout, stderr io.Writer) error {
	registry, err := catalog.LoadRegistry()
	if err != nil {
		return wrapError("load catalog", err)
	}

	stateData, err := loadState(stderr)
	if err != nil {
		return wrapError("load state", err)
	}

	snapshot, err := buildSnapshot(registry, stateData)
	if err != nil {
		return wrapError("build discovery snapshot", err)
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "TOOL\tTIER\tINSTALLED\tOWNERSHIP\tPATH\tVERIFIED"); err != nil {
		return wrapError("write status header", err)
	}

	for _, tool := range registry.Tools() {
		presence := snapshot.Tools[tool.ID]
		verified := "-"
		if presence.Receipt != nil {
			if presence.Receipt.LastVerifyOK {
				verified = "yes"
			} else if presence.Receipt.LastVerifyError != "" {
				verified = "no"
			}
		}

		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%t\t%s\t%s\t%s\n",
			tool.Bin,
			tool.Tier,
			presence.Installed,
			presence.Ownership,
			presence.Path,
			verified,
		); err != nil {
			return wrapError("write status row", err)
		}
	}

	return wrapError("flush status output", writer.Flush())
}

func runCatalog(stdout, stderr io.Writer) error {
	registry, err := catalog.LoadRegistry()
	if err != nil {
		return wrapError("load catalog", err)
	}

	stateData, err := loadState(stderr)
	if err != nil {
		return wrapError("load state", err)
	}

	snapshot, err := buildSnapshot(registry, stateData)
	if err != nil {
		return wrapError("build discovery snapshot", err)
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tBIN\tTIER\tCATEGORY\tINSTALLED"); err != nil {
		return wrapError("write catalog header", err)
	}

	for _, tool := range registry.Tools() {
		presence := snapshot.Tools[tool.ID]
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%t\n",
			tool.Name,
			tool.Bin,
			tool.Tier,
			tool.Category,
			presence.Installed,
		); err != nil {
			return wrapError("write catalog row", err)
		}
	}

	return wrapError("flush catalog output", writer.Flush())
}

func runUpdate(ctx context.Context, stdout, stderr io.Writer, toolID string) error {
	registry, err := catalog.LoadRegistry()
	if err != nil {
		return wrapError("load catalog", err)
	}

	managers := pkgmgr.DetectManagers()
	if len(managers) == 0 {
		return errNoSupportedPackageManagers
	}

	stateData, err := loadState(stderr)
	if err != nil {
		return wrapError("load state", err)
	}

	snapshot, err := buildSnapshot(registry, stateData)
	if err != nil {
		return wrapError("build discovery snapshot", err)
	}

	updatePlan, err := plan.BuildUpdatePlan(snapshot, managers, toolID)
	if err != nil {
		return wrapError("build update plan", err)
	}

	summary, err := plan.ExecuteUpdatePlan(ctx, updatePlan, &stateData, liveVerifier{})
	if err != nil {
		return wrapError("execute update plan", err)
	}

	verified, err := refreshVerifiedTools(ctx, registry, &stateData, liveVerifier{})
	if err != nil {
		return wrapError("refresh verified tools", err)
	}

	if err := state.Save(stateData); err != nil {
		return wrapError("save state", err)
	}

	if err := writeSkillFile(verified); err != nil {
		return wrapError("write skill file", err)
	}

	renderSummary(stdout, "update", summary)

	return nil
}

func runUninstall(ctx context.Context, stdout, stderr io.Writer, toolIDs []string, uninstallAll bool) error {
	registry, err := catalog.LoadRegistry()
	if err != nil {
		return wrapError("load catalog", err)
	}

	managers := pkgmgr.DetectManagers()
	if len(managers) == 0 {
		return errNoSupportedPackageManagers
	}

	stateData, err := loadState(stderr)
	if err != nil {
		return wrapError("load state", err)
	}

	snapshot, err := buildSnapshot(registry, stateData)
	if err != nil {
		return wrapError("build discovery snapshot", err)
	}

	uninstallPlan, err := plan.BuildUninstallPlan(snapshot, managers, toolIDs, uninstallAll)
	if err != nil {
		return wrapError("build uninstall plan", err)
	}

	summary, err := plan.ExecuteUninstallPlan(ctx, uninstallPlan, &stateData)
	if err != nil {
		return wrapError("execute uninstall plan", err)
	}

	verified, err := refreshVerifiedTools(ctx, registry, &stateData, liveVerifier{})
	if err != nil {
		return wrapError("refresh verified tools", err)
	}

	if err := state.Save(stateData); err != nil {
		return wrapError("save state", err)
	}

	if err := writeSkillFile(verified); err != nil {
		return wrapError("write skill file", err)
	}

	renderSummary(stdout, "uninstall", summary)

	return nil
}

func applyShellWorkflow(stdout io.Writer, yes bool, st *state.State, tools []catalog.Tool) error {
	suggestions := shell.Suggestions(tools)
	if len(suggestions) == 0 {
		return nil
	}

	shell.MarkSuggestedSuggestions(suggestions, st)
	if yes {
		printShellSuggestions(stdout, suggestions)

		return nil
	}

	apply, err := confirmApply(stdout)
	if err != nil {
		return wrapError("confirm shell hook application", err)
	}

	if !apply {
		shell.MarkDeclinedSuggestions(suggestions, st)

		return nil
	}

	return wrapError("apply shell hook suggestions", shell.ApplyConfirmedSuggestions(suggestions, st))
}

func confirmApply(stdout io.Writer) (bool, error) {
	if _, err := fmt.Fprint(stdout, "Apply shell hook suggestions now? [y/N]: "); err != nil {
		return false, wrapError("write shell hook prompt", err)
	}

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read shell hook confirmation: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes", nil
}

func printShellSuggestions(stdout io.Writer, suggestions []shell.Suggestion) {
	_, _ = fmt.Fprintln(stdout, "Shell hook suggestions:")
	for _, suggestion := range suggestions {
		_, _ = fmt.Fprintf(stdout, "- %s -> %s\n", suggestion.ToolName, suggestion.InitLine)
	}
}

func loadState(stderr io.Writer) (state.State, error) {
	st, err := state.Load()
	if err != nil && errors.Is(err, state.ErrCorruptState) {
		_, _ = fmt.Fprintln(stderr, "warning: state file is corrupt; continuing with an empty state")

		return st, nil
	}

	return st, wrapError("load state file", err)
}

func buildSnapshot(registry catalog.Registry, st state.State) (discovery.Snapshot, error) {
	paths, err := discovery.ScanPATH(registry.Tools(), exec.LookPath)
	if err != nil {
		return discovery.Snapshot{}, wrapError("scan PATH", err)
	}

	return discovery.Reconcile(registry, st, paths), nil
}

func selectTools(registry catalog.Registry, snapshot discovery.Snapshot, yes bool) ([]catalog.Tool, error) {
	platformTools := registry.ForPlatform(platform.Detect().OS)
	if yes {
		selected := make([]catalog.Tool, 0, len(platformTools))
		for _, tool := range platformTools {
			if tool.DefaultSelected {
				selected = append(selected, tool)
			}
		}

		return selected, nil
	}

	selected, err := tui.RunPicker(platformTools, snapshot)
	if err != nil {
		return nil, wrapError("run interactive picker", err)
	}

	return selected, nil
}

func writeSkillFile(tools []catalog.Tool) error {
	content := skill.Generate(tools)

	return wrapError("persist cli-tools skill", skill.Write(content, skill.DefaultPaths()))
}

func persistVerifiedSkill(
	ctx context.Context,
	registry catalog.Registry,
	st *state.State,
	verifier toolVerifier,
) error {
	verified, err := refreshVerifiedTools(ctx, registry, st, verifier)
	if err != nil {
		return wrapError("refresh verified tools", err)
	}

	if err := state.Save(*st); err != nil {
		return wrapError("save state", err)
	}

	if err := writeSkillFile(verified); err != nil {
		return wrapError("write skill file", err)
	}

	return nil
}

func refreshVerifiedTools(
	ctx context.Context,
	registry catalog.Registry,
	st *state.State,
	verifier toolVerifier,
) ([]catalog.Tool, error) {
	snapshot, err := buildSnapshot(registry, *st)
	if err != nil {
		return nil, wrapError("build discovery snapshot", err)
	}

	verified := make([]catalog.Tool, 0, len(snapshot.Tools))
	for _, tool := range registry.Tools() {
		presence := snapshot.Tools[tool.ID]
		if !presence.Installed {
			if presence.Receipt != nil {
				receipt := *presence.Receipt
				receipt.BinaryPath = ""
				receipt.LastVerifyAt = time.Now().UTC()
				receipt.LastVerifyOK = false
				receipt.LastVerifyError = "binary not found"
				if err := st.SetTool(receipt); err != nil {
					return nil, wrapError("persist missing tool state", err)
				}
			}

			continue
		}

		receipt := toolStateForPresence(presence)
		receipt.Bin = tool.Bin
		receipt.BinaryPath = presence.Path

		result, verifyErr := verifier.Check(ctx, tool)
		receipt.LastVerifyAt = result.CheckedAt
		receipt.LastVerifyOK = result.Verified
		receipt.LastVerifyError = result.Error
		receipt.Version = result.Version
		if verifyErr != nil {
			receipt.LastVerifyOK = false
			receipt.LastVerifyError = verifyErr.Error()
		}

		if err := st.SetTool(receipt); err != nil {
			return nil, wrapError("persist tool verification state", err)
		}

		if verifyErr == nil && result.Verified {
			verified = append(verified, tool)
		}
	}

	return verified, nil
}

func toolStateForPresence(presence discovery.ToolPresence) state.ToolState {
	if presence.Receipt != nil {
		return *presence.Receipt
	}

	return state.ToolState{
		ToolID:     presence.Tool.ID,
		Bin:        presence.Tool.Bin,
		Ownership:  state.OwnershipExternal,
		BinaryPath: presence.Path,
	}
}

func renderSummary(stdout io.Writer, action string, summary plan.Summary) {
	_, _ = fmt.Fprintf(stdout, "%s summary:\n", titleCase(action))
	if len(summary.Installed) > 0 {
		_, _ = fmt.Fprintf(stdout, "  installed: %s\n", strings.Join(summary.Installed, ", "))
	}
	if len(summary.Updated) > 0 {
		_, _ = fmt.Fprintf(stdout, "  updated: %s\n", strings.Join(summary.Updated, ", "))
	}
	if len(summary.Uninstalled) > 0 {
		_, _ = fmt.Fprintf(stdout, "  uninstalled: %s\n", strings.Join(summary.Uninstalled, ", "))
	}
	if len(summary.External) > 0 {
		_, _ = fmt.Fprintf(stdout, "  external: %s\n", strings.Join(summary.External, ", "))
	}
	if len(summary.Skipped) > 0 {
		_, _ = fmt.Fprintf(stdout, "  skipped: %s\n", strings.Join(summary.Skipped, ", "))
	}
	if len(summary.Failed) > 0 {
		failed := make([]string, 0, len(summary.Failed))
		for _, item := range summary.Failed {
			failed = append(failed, fmt.Sprintf("%s (%s)", item.ToolID, item.Error))
		}
		_, _ = fmt.Fprintf(stdout, "  failed: %s\n", strings.Join(failed, ", "))
	}
}

func titleCase(value string) string {
	if value == "" {
		return value
	}

	return strings.ToUpper(value[:1]) + value[1:]
}

func wrapError(action string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", action, err)
}
