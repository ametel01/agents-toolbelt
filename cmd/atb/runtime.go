package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	"github.com/ametel01/agents-toolbelt/internal/plan"
	"github.com/ametel01/agents-toolbelt/internal/platform"
	"github.com/ametel01/agents-toolbelt/internal/selfupdate"
	"github.com/ametel01/agents-toolbelt/internal/shell"
	"github.com/ametel01/agents-toolbelt/internal/skill"
	"github.com/ametel01/agents-toolbelt/internal/state"
	"github.com/ametel01/agents-toolbelt/internal/tui"
	"github.com/ametel01/agents-toolbelt/internal/verify"
)

var (
	errNoSupportedPackageManagers = errors.New("no supported package managers detected")
	errSelfUpdateDevBuild         = errors.New("self-update is unavailable in development builds; use a released atb binary instead")
)

type liveVerifier struct{}

type toolVerifier interface {
	Check(context.Context, catalog.Tool) (verify.VerifyResult, error)
}

type progressManager struct {
	action  plan.ActionType
	index   int
	manager pkgmgr.Manager
	method  catalog.InstallMethod
	stdout  io.Writer
	tool    catalog.Tool
	total   int
}

type reportingVerifier struct {
	next   toolVerifier
	stdout io.Writer
}

type installContext struct {
	registry  catalog.Registry
	stateData state.State
	snapshot  discovery.Snapshot
	managers  []pkgmgr.Manager
	selected  []catalog.Tool
}

func (liveVerifier) Check(ctx context.Context, tool catalog.Tool) (verify.VerifyResult, error) {
	result, err := verify.Check(ctx, tool, verify.ExecExecutor{})
	if err != nil {
		return result, wrapError("verify tool", err)
	}

	return result, nil
}

func runInstall(ctx context.Context, stdout, stderr io.Writer, yes bool) error {
	installCtx, err := prepareInstall(stderr, yes)
	if err != nil {
		return err
	}

	if len(installCtx.selected) == 0 {
		_, writeErr := fmt.Fprintln(stdout, "No tools selected.")

		return wrapError("write empty selection message", writeErr)
	}

	installPlan, err := plan.BuildInstallPlan(installCtx.selected, installCtx.snapshot, installCtx.managers)
	if err != nil {
		return wrapError("build install plan", err)
	}

	if err := renderManagerInfo(stdout, installCtx.managers); err != nil {
		return wrapError("write manager info", err)
	}

	if err := renderPlanPreview(stdout, "install", installPlan); err != nil {
		return wrapError("write install plan preview", err)
	}

	summary, err := plan.ExecuteInstallPlan(
		ctx,
		withProgress(installPlan, stdout),
		&installCtx.stateData,
		reportingVerifier{next: liveVerifier{}, stdout: stdout},
	)
	if err != nil {
		return wrapError("execute install plan", err)
	}

	if err := applyShellWorkflow(stdout, yes, &installCtx.stateData, installCtx.selected); err != nil {
		return wrapError("apply shell workflow", err)
	}

	targets, err := selectTargets(yes)
	if err != nil {
		return wrapError("select skill targets", err)
	}

	if len(targets) == 0 {
		if _, writeErr := fmt.Fprintln(stdout, "Skill generation skipped."); writeErr != nil {
			return wrapError("write skip message", writeErr)
		}
	} else if err := persistVerifiedSkill(ctx, installCtx.registry, &installCtx.stateData, liveVerifier{}, stdout, targets); err != nil {
		return wrapError("persist verified skill", err)
	}

	renderSummary(stdout, "install", summary)

	return nil
}

func prepareInstall(stderr io.Writer, yes bool) (installContext, error) {
	registry, err := catalog.LoadRegistry()
	if err != nil {
		return installContext{}, wrapError("load catalog", err)
	}

	managers := pkgmgr.DetectManagers()
	if len(managers) == 0 {
		return installContext{}, errNoSupportedPackageManagers
	}

	stateData, err := loadState(stderr)
	if err != nil {
		return installContext{}, wrapError("load state", err)
	}

	snapshot, err := buildSnapshot(registry, stateData)
	if err != nil {
		return installContext{}, wrapError("build discovery snapshot", err)
	}

	selected, err := selectTools(registry, snapshot, yes)
	if err != nil {
		return installContext{}, wrapError("select tools", err)
	}

	return installContext{
		registry:  registry,
		stateData: stateData,
		snapshot:  snapshot,
		managers:  managers,
		selected:  selected,
	}, nil
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

func runSelfUpdate(ctx context.Context, stdout, _ io.Writer) error {
	if version == "dev" {
		return wrapError("self update", errSelfUpdateDevBuild)
	}

	result, err := selfupdate.Update(ctx, selfupdate.Options{
		CurrentVersion: version,
		GOARCH:         runtime.GOARCH,
		GOOS:           runtime.GOOS,
	})
	if err != nil {
		return wrapError("self update", err)
	}

	if !result.Updated {
		if _, err := fmt.Fprintf(stdout, "atb is already up to date (%s)\n", result.CurrentVersion); err != nil {
			return wrapError("write self update status", err)
		}

		return nil
	}

	if _, err := fmt.Fprintf(stdout, "updated atb from %s to %s\n", result.CurrentVersion, result.LatestVersion); err != nil {
		return wrapError("write self update result", err)
	}
	if _, err := fmt.Fprintf(stdout, "binary path: %s\n", result.ExecutablePath); err != nil {
		return wrapError("write self update path", err)
	}

	return nil
}

func runToolUpdate(ctx context.Context, stdout, stderr io.Writer, toolID string) error {
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

	if err := renderManagerInfo(stdout, managers); err != nil {
		return wrapError("write manager info", err)
	}

	if err := renderPlanPreview(stdout, "update", updatePlan); err != nil {
		return wrapError("write update plan preview", err)
	}

	summary, err := plan.ExecuteUpdatePlan(
		ctx,
		withProgress(updatePlan, stdout),
		&stateData,
		reportingVerifier{next: liveVerifier{}, stdout: stdout},
	)
	if err != nil {
		return wrapError("execute update plan", err)
	}

	if err := persistVerifiedSkill(ctx, registry, &stateData, liveVerifier{}, stdout, skill.AllTargets()); err != nil {
		return wrapError("persist verified skill", err)
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

	if err := renderManagerInfo(stdout, managers); err != nil {
		return wrapError("write manager info", err)
	}

	if err := renderPlanPreview(stdout, "uninstall", uninstallPlan); err != nil {
		return wrapError("write uninstall plan preview", err)
	}

	summary, err := plan.ExecuteUninstallPlan(ctx, withProgress(uninstallPlan, stdout), &stateData)
	if err != nil {
		return wrapError("execute uninstall plan", err)
	}

	if err := persistVerifiedSkill(ctx, registry, &stateData, liveVerifier{}, stdout, skill.AllTargets()); err != nil {
		return wrapError("persist verified skill", err)
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

func selectTargets(yes bool) ([]skill.Target, error) {
	if yes {
		return skill.AllTargets(), nil
	}

	selected, err := tui.RunTargetPicker(skill.AllTargets())
	if err != nil {
		return nil, wrapError("run target picker", err)
	}

	return selected, nil
}

func persistVerifiedSkill(
	ctx context.Context,
	registry catalog.Registry,
	st *state.State,
	verifier toolVerifier,
	stdout io.Writer,
	targets []skill.Target,
) error {
	verified, err := refreshVerifiedTools(ctx, registry, st, verifier)
	if err != nil {
		return wrapError("refresh verified tools", err)
	}

	paths, err := skill.PathsForTargets(targets)
	if err != nil {
		return wrapError("resolve skill paths", err)
	}

	if _, err := fmt.Fprintf(stdout, "Generating cli-tools skills for %d verified tools:\n", len(verified)); err != nil {
		return wrapError("write skill generation header", err)
	}
	for _, path := range paths {
		if _, err := fmt.Fprintf(stdout, "  - %s\n", path); err != nil {
			return wrapError("write skill generation path", err)
		}
	}

	if err := state.Save(*st); err != nil {
		return wrapError("save state", err)
	}

	content := skill.Generate(verified)
	if err := skill.Write(content, paths); err != nil {
		return wrapError("write skill file", err)
	}

	if _, err := fmt.Fprintln(stdout, "Skill generation complete."); err != nil {
		return wrapError("write skill generation footer", err)
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return wrapError("write skill generation spacing", err)
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

func renderPlanPreview(stdout io.Writer, operation string, lifecyclePlan plan.Plan) error {
	if _, err := fmt.Fprintf(stdout, "%s plan:\n", titleCase(operation)); err != nil {
		return wrapError("write plan preview header", err)
	}

	for _, action := range lifecyclePlan.Actions {
		if _, err := fmt.Fprintf(stdout, "  - %s\n", describeAction(action)); err != nil {
			return wrapError("write plan preview action", err)
		}
	}

	_, err := fmt.Fprintln(stdout)

	return wrapError("write plan preview spacing", err)
}

func describeAction(action plan.Action) string {
	name := actionLabel(action.Tool)
	managerNote := methodNote(action.Method.Manager)

	switch action.Type {
	case plan.ActionInstall:
		return fmt.Sprintf("install %s via %s%s", name, action.Method.Manager, managerNote)
	case plan.ActionUpdate:
		return fmt.Sprintf("update %s via %s%s", name, action.Method.Manager, managerNote)
	case plan.ActionUninstall:
		return fmt.Sprintf("uninstall %s via %s%s", name, action.Method.Manager, managerNote)
	case plan.ActionAlreadyInstalled:
		return fmt.Sprintf("skip %s (%s)", name, action.Reason)
	case plan.ActionSkip:
		return fmt.Sprintf("skip %s (%s)", name, action.Reason)
	default:
		return fmt.Sprintf("%s %s", action.Type, name)
	}
}

func withProgress(lifecyclePlan plan.Plan, stdout io.Writer) plan.Plan {
	actions := slices.Clone(lifecyclePlan.Actions)
	total := 0
	for _, action := range actions {
		if isExecutable(action.Type) {
			total++
		}
	}

	index := 0
	for actionIndex, action := range actions {
		if !isExecutable(action.Type) {
			continue
		}

		index++
		actions[actionIndex].Manager = progressManager{
			action:  action.Type,
			index:   index,
			manager: action.Manager,
			method:  action.Method,
			stdout:  stdout,
			tool:    action.Tool,
			total:   total,
		}
	}

	return plan.Plan{Actions: actions}
}

func isExecutable(actionType plan.ActionType) bool {
	switch actionType {
	case plan.ActionInstall, plan.ActionUpdate, plan.ActionUninstall:
		return true
	default:
		return false
	}
}

func (m progressManager) Name() string {
	return m.manager.Name()
}

func (m progressManager) Available() bool {
	return m.manager.Available()
}

func (m progressManager) Install(ctx context.Context, method catalog.InstallMethod) error {
	return m.run(ctx, method, "Installing")
}

func (m progressManager) Update(ctx context.Context, method catalog.InstallMethod) error {
	return m.run(ctx, method, "Updating")
}

func (m progressManager) Uninstall(ctx context.Context, method catalog.InstallMethod) error {
	return m.run(ctx, method, "Uninstalling")
}

func (m progressManager) run(ctx context.Context, method catalog.InstallMethod, verb string) error {
	_, _ = fmt.Fprintf(
		m.stdout,
		"[%d/%d] %s %s via %s...\n",
		m.index,
		m.total,
		verb,
		actionLabel(m.tool),
		method.Manager,
	)

	var err error
	switch m.action {
	case plan.ActionInstall:
		err = m.manager.Install(ctx, method)
	case plan.ActionUpdate:
		err = m.manager.Update(ctx, method)
	case plan.ActionUninstall:
		err = m.manager.Uninstall(ctx, method)
	}

	if err != nil {
		_, _ = fmt.Fprintf(m.stdout, "      failed %s: %v\n", actionLabel(m.tool), err)

		return wrapError(fmt.Sprintf("%s %s via %s", strings.ToLower(verb), actionLabel(m.tool), method.Manager), err)
	}

	_, _ = fmt.Fprintf(m.stdout, "      completed %s\n", actionLabel(m.tool))

	return nil
}

func (v reportingVerifier) Check(ctx context.Context, tool catalog.Tool) (verify.VerifyResult, error) {
	_, _ = fmt.Fprintf(v.stdout, "      verifying %s...\n", actionLabel(tool))

	result, err := v.next.Check(ctx, tool)
	if err != nil {
		_, _ = fmt.Fprintf(v.stdout, "      verification failed for %s: %v\n", actionLabel(tool), err)

		return result, wrapError(fmt.Sprintf("verify %s", actionLabel(tool)), err)
	}

	if !result.Verified {
		message := result.Error
		if message == "" {
			message = "verification returned false"
		}
		_, _ = fmt.Fprintf(v.stdout, "      verification failed for %s: %s\n", actionLabel(tool), message)

		return result, nil
	}

	if result.Version != "" {
		_, _ = fmt.Fprintf(v.stdout, "      verified %s (%s)\n", actionLabel(tool), result.Version)
	} else {
		_, _ = fmt.Fprintf(v.stdout, "      verified %s\n", actionLabel(tool))
	}

	return result, nil
}

func actionLabel(tool catalog.Tool) string {
	if tool.Name != "" {
		return tool.Name
	}

	return tool.Bin
}

func renderManagerInfo(stdout io.Writer, managers []pkgmgr.Manager) error {
	names := make([]string, 0, len(managers))
	secondary := make([]string, 0, len(managers))
	for _, manager := range managers {
		name := manager.Name()
		names = append(names, name)
		if isSecondaryManager(name) {
			secondary = append(secondary, name)
		}
	}

	if _, err := fmt.Fprintf(stdout, "Detected package managers: %s\n", strings.Join(names, ", ")); err != nil {
		return wrapError("write package manager summary", err)
	}

	if len(secondary) > 0 {
		if _, err := fmt.Fprintf(
			stdout,
			"Note: %s must already be installed on the host; atb does not bootstrap those managers.\n\n",
			strings.Join(secondary, ", "),
		); err != nil {
			return wrapError("write secondary manager note", err)
		}

		return nil
	}

	_, err := fmt.Fprintln(stdout)

	return wrapError("write package manager spacing", err)
}

func methodNote(manager string) string {
	if !isSecondaryManager(manager) {
		return ""
	}

	return " (requires that manager on the host)"
}

func isSecondaryManager(manager string) bool {
	switch manager {
	case "cargo", "go", "pipx":
		return true
	default:
		return false
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
