# Implementation Plan: agents-toolbelt (`atb`)

> Each step is atomic and verifiable. No step depends on unvalidated work.
> Estimated steps: 15. Each step ends with a concrete validation gate.

---

## Step 0 — Project Scaffold & Build Infrastructure

Status: **COMPLETED**

**Goal**: Go module compiles, linter runs, `make verify` works on an empty project.

**Files to create**:
- `go.mod` — module `github.com/ametel01/agents-toolbelt`, go 1.24
- `cmd/atb/main.go` — minimal `func main()` that prints version
- `internal/logx/logx.go` — minimal shared logging and user-facing error helpers, expanded in later steps
- `.gitignore` — Go binaries, vendor, IDE files
- `.golangci.yml` — linter config (enable govet, staticcheck, errcheck, unused, gosimple)
- `Makefile` — single `verify` target wiring all quality gates
- `.goreleaser.yaml` — cross-compile config for linux/darwin amd64/arm64
- `CHANGELOG.md` — Keep a Changelog 1.1.0 skeleton with `## [Unreleased]`

**Makefile targets**:
```makefile
.PHONY: verify fmt vet lint test build vulncheck

verify: fmt vet lint test build vulncheck

fmt:
	gofmt -l . | grep -q . && exit 1 || true

vet:
	go vet ./...

lint:
	staticcheck ./...
	golangci-lint run

test:
	go test ./...
	go test -race ./...

build:
	go build ./...

vulncheck:
	govulncheck ./...
```

**Validation**:
```bash
go build ./cmd/atb && ./atb        # prints version string
make verify                         # all gates pass (trivially, no code yet)
```

**Failure modes**:
- Go 1.24 not installed → `go version` check in Makefile preamble
- `staticcheck`/`golangci-lint`/`govulncheck` not installed → document in README or add `tools.go` with `//go:build tools` for tool deps

**CHANGELOG**: Create the file in this step, but do not add an `Unreleased` entry unless the scaffold introduces shipped functional behavior. Changelog entries are for functional or user-visible changes only.

---

## Step 1 — Cobra CLI Skeleton with Subcommands

Status: **COMPLETED**

**Goal**: `atb install`, `atb status`, `atb update`, `atb uninstall`, `atb catalog` all parse and print "not implemented".

**Files to create/modify**:
- `cmd/atb/main.go` — wire root command
- `cmd/atb/root.go` — root cobra command with `--version` flag
- `cmd/atb/install.go` — `install` subcommand with `-y`/`--yes` flag
- `cmd/atb/status.go` — `status` subcommand
- `cmd/atb/update.go` — `update` subcommand (accepts optional tool name arg)
- `cmd/atb/uninstall.go` — `uninstall` subcommand (requires tool name or `--all`)
- `cmd/atb/catalog.go` — `catalog` subcommand

**Key snippet** (`cmd/atb/root.go`):
```go
var rootCmd = &cobra.Command{
    Use:     "atb",
    Short:   "Install and manage CLI tools for coding agent workflows",
    Version: version, // injected at build via ldflags
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

**Validation**:
```bash
go build -o atb ./cmd/atb
./atb --version
./atb install --help   # shows -y flag
./atb catalog           # prints "not implemented"
make verify
```

**Failure modes**:
- Cobra API misuse → caught by `go build`
- Missing flag wiring → manual `--help` check

**CHANGELOG**: Usually no changelog entry. Placeholder command parsing without functional behavior is not release-note material.

---

## Step 2 — Embedded Tool Registry (`internal/catalog`)

**Goal**: 22 tools defined in JSON, embedded in binary, loaded and validated at startup.

**Files to create**:
- `internal/catalog/registry.json` — full 22-tool registry (schema from spec)
- `internal/catalog/catalog.go` — types, embed, load, validate, lookup functions
- `internal/catalog/catalog_test.go` — validate all 22 entries parse, bins unique, tiers valid, every tool has ≥1 install method, every tool has a verify command

**Type definitions** (`catalog.go`):
```go
type Tool struct {
    ID              string          `json:"id"`
    Bin             string          `json:"bin"`
    Name            string          `json:"name"`
    Tier            Tier            `json:"tier"`            // must|should|nice
    Category        string          `json:"category"`
    Description     string          `json:"description"`
    Platforms       []string        `json:"platforms"`
    InstallMethods  []InstallMethod `json:"install_methods"`
    ShellHook       string          `json:"shell_hook"`      // none|optional|required
    Auth            string          `json:"auth"`            // none|optional|required
    ServiceDep      string          `json:"service_dependency"`
    Interactive     bool            `json:"interactive"`
    TUI             bool            `json:"tui"`
    Verify          VerifySpec      `json:"verify"`
    SkillExpose     bool            `json:"skill_expose"`
    DefaultSelected bool            `json:"default_selected"`
    Tags            []string        `json:"tags"`
}

type InstallMethod struct {
    Manager          string   `json:"manager"`
    Package          string   `json:"package"`
    Command          []string `json:"command"`
    UpdateCommand    []string `json:"update_command"`
    UninstallCommand []string `json:"uninstall_command"`
    RequiresSudo     bool     `json:"requires_sudo"`
    TimeoutSeconds   int      `json:"timeout_seconds"`
}

type VerifySpec struct {
    Command           []string `json:"command"`
    TimeoutSeconds    int      `json:"timeout_seconds"`
    ExpectedExitCodes []int    `json:"expected_exit_codes"`
    VersionRegex      string   `json:"version_regex,omitempty"`
}
```

**Validation functions to test**:
- `LoadRegistry()` returns 22 tools, no error
- Every `tool.Bin` is non-empty and unique
- Every `tool.Tier` is one of `must`, `should`, `nice`
- Every tool has at least one `InstallMethod`
- Every `VerifySpec.Command` is non-empty
- `ByID("fzf")` returns the fzf tool
- `ByTier("must")` returns only `must` tools and a non-zero result
- `ByCategory("navigation")` returns tools whose category is `navigation`
- Platform filtering works (all 22 have linux+macos)

**Validation**:
```bash
go test ./internal/catalog/... -v    # all registry tests pass
make verify
```

**Failure modes**:
- JSON schema drift → test catches missing fields
- Duplicate bin names → test catches
- Typo in tier value → type validation rejects

**CHANGELOG**: Add an entry only when the embedded catalog becomes functional behavior exposed by the binary.

---

## Step 3 — Platform & Package Manager Detection (`internal/pkgmgr`)

**Goal**: Detect OS/arch and available package managers, pick best match for a tool.

**Files to create**:
- `internal/platform/platform.go` — OS and arch detection (`runtime.GOOS`, `runtime.GOARCH`)
- `internal/platform/platform_test.go`
- `internal/pkgmgr/detect.go` — scan PATH for known managers, return ordered list
- `internal/pkgmgr/manager.go` — `Manager` interface + priority ordering
- `internal/pkgmgr/<manager>.go` — one adapter per supported manager (`brew`, `apt`, `dnf`, `pacman`, `snap`, `go`, `pipx`, `cargo`)
- `internal/pkgmgr/detect_test.go` — test with fake PATH

**Interface** (`manager.go`):
```go
type Manager interface {
    Name() string
    Available() bool
    Install(ctx context.Context, method catalog.InstallMethod) error
    Update(ctx context.Context, method catalog.InstallMethod) error
    Uninstall(ctx context.Context, method catalog.InstallMethod) error
}
```

**Detection logic** (`detect.go`):
```go
// Priority: brew > apt > dnf > pacman > snap > go > pipx > cargo
var managerPriority = []string{"brew", "apt", "dnf", "pacman", "snap", "go", "pipx", "cargo"}

func DetectManagers() []Manager { ... }

func SelectBest(tool catalog.Tool, available []Manager) (catalog.InstallMethod, Manager, error) {
    // For each available manager in priority order,
    // find first matching install_method in the tool
}
```

**Validation**:
```bash
go test ./internal/pkgmgr/... -v
go test ./internal/platform/... -v
make verify
```

**Failure modes**:
- No package manager found → `SelectBest` returns descriptive error
- Tool has no method for available managers → returns specific `ErrNoMatchingMethod`

**CHANGELOG**: Add an entry only when package manager detection/adapters become functional behavior.

---

## Step 4 — Discovery & Reconciliation (`internal/discovery`)

**Goal**: Reconcile catalog, PATH detection, and persisted state into a single authoritative runtime view used by all commands.

**Files to create**:
- `internal/discovery/discovery.go` — PATH scanning and state reconciliation
- `internal/discovery/discovery_test.go` — tests for managed/external/missing classification and corrupt-state recovery behavior

**Key types**:
```go
type ToolPresence struct {
    Tool           catalog.Tool
    Path           string
    Installed      bool
    Ownership      string // "managed" | "external" | "missing"
    Receipt        *state.ToolState
    VerifyResult   *verify.VerifyResult
}

type Snapshot struct {
    Tools map[string]ToolPresence
}

func ScanPATH(tools []catalog.Tool, lookPath LookPather) (map[string]string, error)

func Reconcile(reg catalog.Registry, st state.State, paths map[string]string) Snapshot
```

**Logic**:
1. Scan PATH using `exec.LookPath`/equivalent for every catalog binary
2. Merge scan results with persisted receipts from `state.json`
3. Classify each tool as `managed`, `external`, or `missing`
4. Preserve state ownership authority: a PATH hit without a receipt is `external`, not `managed`
5. Surface path, receipt metadata, and last verification data in one structure reused by `install`, `status`, `catalog`, `update`, and `uninstall`

**Test cases**:
- PATH hit + managed receipt → `managed`
- PATH hit + no receipt → `external`
- No PATH hit + managed receipt → installed state is stale but ownership remains `managed` until uninstall/reinstall reconciliation logic handles it
- Corrupt or missing state file still yields a valid snapshot from PATH discovery

**Validation**:
```bash
go test ./internal/discovery/... -v
make verify
```

**CHANGELOG**: Add an entry only when reconciliation behavior becomes user-visible through command output.

---

## Step 5 — Safe Command Execution (`internal/execx`)

**Goal**: Run external commands with timeouts, stdout/stderr capture, no shell invocation.

**Files to create**:
- `internal/execx/exec.go` — `Run(ctx, args []string, timeout time.Duration) (Result, error)`
- `internal/execx/exec_test.go` — test with real commands (`echo`, `false`, timeout)

**Key type**:
```go
type Result struct {
    ExitCode int
    Stdout   string
    Stderr   string
    Duration time.Duration
}

func Run(ctx context.Context, args []string, timeout time.Duration) (Result, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    cmd := exec.CommandContext(ctx, args[0], args[1:]...)
    // capture stdout, stderr
    // return Result with exit code
}
```

**Critical rules**:
- Never use `sh -c` — always `exec.Command(args[0], args[1:]...)`
- Context timeout enforced — kills process on timeout
- Return exit code even on non-zero (don't treat as Go error)

**Validation**:
```bash
go test ./internal/execx/... -v -race
make verify
```

**Failure modes**:
- Command not found → `exec.ErrNotFound`, wrapped
- Timeout → `context.DeadlineExceeded`, wrapped with "command timed out"
- Signal killed → detect and report

**CHANGELOG**: Add an entry only when external command execution powers shipped behavior.

---

## Step 6 — Tool Verification (`internal/verify`)

**Goal**: Run a tool's verify command and determine if it's functional.

**Files to create**:
- `internal/verify/verify.go` — `Check(ctx, tool catalog.Tool, executor) (VerifyResult, error)`
- `internal/verify/verify_test.go` — test with mock executor

**Key type**:
```go
type VerifyResult struct {
    ToolID    string
    Found     bool       // command -v found it
    Verified  bool       // verify command succeeded
    Version   string     // extracted version if regex provided
    Error     string     // error message if failed
    CheckedAt time.Time
}

func Check(ctx context.Context, tool catalog.Tool, exec execx.Executor) (VerifyResult, error) {
    // 1. Check if binary exists via LookPath
    // 2. Run tool.Verify.Command with timeout
    // 3. Check exit code against tool.Verify.ExpectedExitCodes
    // 4. Extract version via tool.Verify.VersionRegex if set
}
```

**Inject executor interface** so tests use fakes:
```go
type Executor interface {
    Run(ctx context.Context, args []string, timeout time.Duration) (execx.Result, error)
}
```

**Validation**:
```bash
go test ./internal/verify/... -v
make verify
```

**Failure modes**:
- Binary not on PATH → `Found: false`
- Verify command times out → `Verified: false, Error: "timed out"`
- Unexpected exit code → `Verified: false`
- Version regex doesn't match → `Version: ""` but still `Verified: true` if exit code OK

**CHANGELOG**: Add an entry only when verification is wired into functional commands.

---

## Step 7 — State Management (`internal/state`)

**Goal**: Read/write `~/.config/atb/state.json`, track install receipts and ownership.

**Files to create**:
- `internal/state/state.go` — types, Load, Save, receipt CRUD
- `internal/state/state_test.go` — round-trip tests, corruption recovery, ownership rules

**Schema** (`state.go`):
```go
type State struct {
    Version    int                `json:"version"` // schema version for migrations
    Tools      map[string]ToolState `json:"tools"`  // keyed by tool ID
    LastRunAt  time.Time          `json:"last_run_at"`
}

type ToolState struct {
    ToolID                 string            `json:"tool_id"`
    Bin                    string            `json:"bin"`
    Ownership              string            `json:"ownership"` // "managed" | "external"
    InstallManager         string            `json:"install_manager,omitempty"`
    InstallPackage         string            `json:"install_package,omitempty"`
    InstallCommand         []string          `json:"install_command,omitempty"`
    UpdateCommand          []string          `json:"update_command,omitempty"`
    UninstallCommand       []string          `json:"uninstall_command,omitempty"`
    InstalledAt            time.Time         `json:"installed_at,omitempty"`
    LastUpdateAttemptAt    time.Time         `json:"last_update_attempt_at,omitempty"`
    LastVerifyAt           time.Time         `json:"last_verify_at,omitempty"`
    LastVerifyOK           bool              `json:"last_verify_ok"`
    LastVerifyError        string            `json:"last_verify_error,omitempty"`
    Version                string            `json:"version,omitempty"`
    ShellHookStatus        string            `json:"shell_hook_status,omitempty"` // "pending"|"suggested"|"applied"|"declined"
    ShellHookSuggestedAt   time.Time         `json:"shell_hook_suggested_at,omitempty"`
    ShellHookAppliedAt     time.Time         `json:"shell_hook_applied_at,omitempty"`
    BinaryPath             string            `json:"binary_path,omitempty"`
    Metadata               map[string]string `json:"metadata,omitempty"`
}
```

**Critical rules**:
- `mkdir -p` the config dir on first write
- Atomic write (write to temp file, rename) to avoid corruption
- If state file is missing → return empty state, not error
- If state file is corrupt JSON → warn, return empty state, do not crash
- State ownership is authoritative for uninstall/update permissions
- Successful `atb` installs create a managed receipt immediately, even if later verification fails
- Verification result is stored separately from install ownership so failed verification does not orphan an `atb`-installed tool

**Test cases**:
- Round-trip: save then load → identical
- Missing file → empty state
- Corrupt JSON → empty state + warning
- AddReceipt sets ownership to "managed"
- MarkExternal sets ownership to "external"
- Tool with "external" ownership → `IsATBManaged()` returns false
- Successful install + failed verification still persists a managed receipt
- Commands used in the chosen install method are persisted in the receipt

**Validation**:
```bash
go test ./internal/state/... -v -race
make verify
```

**CHANGELOG**: Add an entry only when persistent state affects shipped behavior.

---

## Step 8 — Install/Update/Uninstall Planning (`internal/plan`)

**Goal**: Given catalog + reconciled discovery snapshot + detected managers, produce ordered plans for install, update, and uninstall.

**Files to create**:
- `internal/plan/plan.go` — `Plan` type and `BuildInstallPlan` function
- `internal/plan/update.go` — `BuildUpdatePlan`
- `internal/plan/uninstall.go` — `BuildUninstallPlan`
- `internal/plan/plan_test.go`

**Key types**:
```go
type Action struct {
    Tool          catalog.Tool
    Type          ActionType         // Install, Skip, AlreadyInstalled
    Method        catalog.InstallMethod
    Manager       pkgmgr.Manager
    Reason        string             // why skipped, etc.
}

type Plan struct {
    Actions []Action
}

func BuildInstallPlan(
    selected []catalog.Tool,
    snapshot discovery.Snapshot,
    managers []pkgmgr.Manager,
) (Plan, error)
```

**Logic**:
1. For each selected tool:
   - If present in snapshot as "managed" → `AlreadyInstalled`
   - If present in snapshot as "external" → `AlreadyInstalled` (external)
   - If not on PATH → find best manager match → `Install`
   - If no manager match → `Skip` with reason
2. Order: `must` tier first, then `should`, then `nice`
3. `BuildUpdatePlan`:
   - include only tools with managed receipts
   - re-select an update method from currently available managers
   - skip with reason if no longer supported by any detected manager
4. `BuildUninstallPlan`:
   - include only tools with managed receipts
   - refuse external tools and missing tools without a managed receipt

**Test cases**:
- All tools missing, brew available → all get Install actions
- Tool already on PATH, no receipt → AlreadyInstalled + external
- Tool already on PATH, managed receipt → AlreadyInstalled + managed
- No matching manager → Skip with clear reason
- Mixed scenario with must/should/nice ordering
- Update plan includes managed tools only
- Uninstall plan refuses external tools

**Validation**:
```bash
go test ./internal/plan/... -v
make verify
```

**CHANGELOG**: Add an entry only when planning logic changes observable command behavior.

---

## Step 9 — Plan Execution (Install/Update/Uninstall)

**Goal**: Execute a plan: install tools, verify, update state.

**Files to create**:
- `internal/plan/executor.go` — `ExecutePlan(ctx, plan, state, verifier) (Summary, error)`
- `internal/plan/executor_test.go` — integration test with fake executor + fake managers

**Key type**:
```go
type Summary struct {
    Installed []string
    Skipped   []string
    Failed    []FailedTool
    External  []string
}

type FailedTool struct {
    ToolID string
    Error  string
}
```

**Logic per Install action**:
1. Call `manager.Install(ctx, method)`
2. On failure → log, add to `Failed`, continue
3. On success → persist/update managed receipt immediately with manager, package, commands used, and install timestamp
4. Run `verify.Check()`
5. If verified → update verification fields, add to `Installed`
6. If verify fails → keep managed receipt, record failed verification fields, add to `Failed` with "installed but verification failed"

**Same pattern for update/uninstall** (separate functions):
- `ExecuteUpdatePlan` — only for "managed" tools, records `LastUpdateAttemptAt`, re-verifies after update, preserves receipt on verify failure
- `ExecuteUninstallPlan` — only for "managed" tools, refuses external, removes receipt only after successful uninstall execution

**Test cases**:
- Happy path: 3 tools install and verify
- One tool fails install → others still succeed
- Tool installs but verify fails → marked failed, not in skill, still uninstallable because receipt exists
- Update on external tool → refused
- Uninstall on external tool → refused
- Ctrl+C mid-install → context cancellation, partial summary returned

**Validation**:
```bash
go test ./internal/plan/... -v -race
make verify
```

**CHANGELOG**: Add an entry only when execution behavior changes shipped command semantics.

---

## Step 10 — Skill Generation (`internal/skill`)

**Goal**: Generate `SKILL.md` to both Claude Code and Codex paths from verified tools.

**Files to create**:
- `internal/skill/skill.go` — `Generate(tools []catalog.Tool) string`, `Write(content, paths) error`, and `DefaultPaths() []string`
- `internal/skill/skill_test.go` — golden test: given known tools, output matches expected markdown
- `internal/skill/testdata/golden_skill.md` — expected golden output

**Template output**:
```markdown
---
name: cli-tools
description: >-
  Use when working with CLI tools in the terminal. Lists verified CLI tools
  available on this system, grouped by category. Activate when a task involves
  terminal commands, file operations, API calls, or development workflows.
---

# CLI Tools Inventory

## Available Tools

### Navigation
- `fzf`
- `zoxide`

### File Viewing
- `bat`

[...grouped by category, only verified tools...]
```

**Rules**:
- Only include tools where `SkillExpose: true` AND verified
- Group by category, sorted alphabetically within group
- Category order follows the spec's category model
- No descriptions, no flags, no examples
- `mkdir -p` output dirs before writing

**Output paths**:
- `~/.claude/skills/cli-tools/SKILL.md`
- `~/.agents/skills/cli-tools/SKILL.md`

**Golden test**: snapshot the output of `Generate()` with a fixed set of 5 tools and compare byte-for-byte.

**Validation**:
```bash
go test ./internal/skill/... -v
make verify
```

**Failure modes**:
- Permission denied on write path → return error, don't crash
- No verified tools → generate valid SKILL.md with empty tools section

**CHANGELOG**: Add an entry only when generated skill output changes functionally.

---

## Step 11 — Shell Integration (`internal/shell`)

**Goal**: Detect user shell, render init-line suggestions, and optionally apply confirmed rc-file edits idempotently for tools that need shell hooks.

**Files to create**:
- `internal/shell/shell.go` — `DetectShell() string`, `Suggestions(tools) []Suggestion`, `ApplyConfirmedSuggestions(...)`
- `internal/shell/shell_test.go`

**Types**:
```go
type Suggestion struct {
    ToolName string
    Shell    string
    RCFile   string   // e.g. ~/.zshrc
    InitLine string   // e.g. eval "$(zoxide init zsh)"
    Required bool     // shell_hook == "required"
}
```

**Shell detection**: `$SHELL` env var → basename → zsh, bash, fish.

**Tools with shell hooks** (from catalog):
- `zoxide` — required — `eval "$(zoxide init <shell>)"`
- `atuin` — required — `eval "$(atuin init <shell>)"`
- `direnv` — required — `eval "$(direnv hook <shell>)"`
- `starship` — optional — `eval "$(starship init <shell>)"`

**Critical rules**:
- Never modify rc files without explicit user confirmation
- Suggested/apply state is persisted in `state.json`
- Applying changes must be idempotent: do not duplicate init lines already present
- If the user declines, record that result and continue without failure

**Apply flow**:
1. Render suggestions for tools with `shell_hook: required` or `optional`
2. Prompt the user to apply, skip, or show all
3. If confirmed, append missing init lines to the detected rc file only once
4. Update shell hook state in `state.json`

**Validation**:
```bash
go test ./internal/shell/... -v
make verify
```

**CHANGELOG**: Add an entry only when shell suggestion/apply behavior changes user-visible functionality.

---

## Step 12 — TUI Picker (`internal/tui`)

**Goal**: Interactive bubbletea picker with categories, tiers, search, and toggle.

**Files to create**:
- `internal/tui/picker.go` — bubbletea Model for tool selection
- `internal/tui/styles.go` — lipgloss styles
- `internal/tui/picker_test.go` — test model updates (key presses → state changes)

**Picker behavior**:
- Tools grouped by category
- `must` tier → preselected checkbox
- `should` tier → shown, unchecked
- `nice` tier → collapsed under "▸ More tools (2)" toggle
- Already-installed tools → marked with ✓, still toggleable
- Keybindings: `space` toggle, `a` select all, `n` deselect all, `enter` confirm, `q`/`esc` quit, `/` search
- Returns `[]catalog.Tool` of selected tools

**Test approach**: send keypress messages to the model's `Update()`, assert on resulting state (selected tools, expanded sections). No visual rendering tests needed.

**Validation**:
```bash
go test ./internal/tui/... -v
make verify
```

**Failure modes**:
- User selects nothing → return empty slice, caller handles gracefully
- Terminal too small → lipgloss handles wrapping

**CHANGELOG**: Add an entry only when TUI behavior changes functionally.

---

## Step 13 — Wire Commands to Business Logic

**Goal**: Connect cobra commands to internal packages. All 5 commands functional.

**Files to modify**:
- `cmd/atb/install.go` — wire: detect platform → detect managers → load state → discover/reconcile PATH → (TUI or headless) → plan → execute → save state → generate skill → shell suggestions/apply prompt → summary
- `cmd/atb/status.go` — wire: load catalog → load state → discover/reconcile PATH → optionally re-verify known/present tools → render table
- `cmd/atb/update.go` — wire: load state → filter managed → re-detect managers → build update plan → execute → re-verify → save state → regenerate skill
- `cmd/atb/uninstall.go` — wire: load state → discover/reconcile → build uninstall plan → execute uninstall → save state → regenerate skill
- `cmd/atb/catalog.go` — wire: load catalog → load state → discover/reconcile PATH → render table including installed status

**Key rule**: Cobra handlers should be thin — 20-30 lines max. All logic lives in internal packages.

**`install` command flow** (pseudocode):
```go
func runInstall(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    reg := catalog.LoadRegistry()
    plat := platform.Detect()
    mgrs := pkgmgr.DetectManagers()
    st := state.Load()
    paths, err := discovery.ScanPATH(reg.Tools(), exec.LookPath)
    if err != nil {
        return err
    }
    snap := discovery.Reconcile(reg, st, paths)

    var selected []catalog.Tool
    if headless {
        selected = reg.DefaultSelected(plat)
    } else {
        selected = tui.RunPicker(reg, snap, plat)
    }
    if len(selected) == 0 {
        fmt.Println("No tools selected.")
        return nil
    }

    p := plan.BuildInstallPlan(selected, snap, mgrs)
    summary := plan.ExecuteInstallPlan(ctx, p, st, verify.NewVerifier())
    state.Save(st)
    skill.Write(skill.Generate(st.VerifiedTools()), skill.DefaultPaths())
    shell.MaybeApplySuggestions(st.ToolsNeedingShellHook())
    printSummary(summary)
    return nil
}
```

**Validation**:
```bash
go build -o atb ./cmd/atb
./atb catalog                    # renders table of 22 tools
./atb status                     # shows installed/missing for all tools
./atb install --help             # shows -y flag
# Manual test: ./atb install -y  # (on a real system, installs default tools)
make verify
```

**Failure modes**:
- No package managers detected → clear error message with install instructions
- State file permission denied → error message, don't crash
- Corrupt state file → warn, rebuild snapshot from PATH where safe
- Externally managed tool present on PATH → visible in picker/status/catalog, never uninstallable by `atb`

**CHANGELOG**: Add an entry when the commands become functionally usable end to end.

---

## Step 14 — Integration Tests

**Goal**: End-to-end tests with fakes validating full install/update/uninstall cycles.

**Files to create**:
- `internal/plan/integration_test.go` — full cycle with fake executor and managers
- `internal/skill/integration_test.go` — verify skill output from realistic state
- `internal/state/integration_test.go` — idempotency: install twice → same state

**Test scenarios**:
1. **Fresh install**: 5 tools selected → all install → all verify → state has 5 managed entries → skill lists 5 tools
2. **Partial failure**: 3 tools selected, 1 install fails → 2 in state, 2 in skill, summary shows 1 failed
3. **Idempotency**: run install plan twice → no duplicate receipts, no re-installs
4. **External tool**: tool on PATH but no receipt → marked external → not uninstallable
5. **Update managed only**: 2 managed + 1 external → update runs on 2, skips 1
6. **Uninstall**: remove managed tool → state cleared → skill regenerated without it
7. **Uninstall external refused**: attempt uninstall on external → error, no action
8. **Install succeeds, verify fails**: managed receipt persists, skill excludes tool, uninstall still works
9. **Shell apply confirmation**: confirmed apply writes exactly one init line; decline records status without editing rc file
10. **Catalog/status reconciliation**: PATH-only tool appears as external and installed in both commands

**Validation**:
```bash
go test ./... -v -race -count=1
make verify
```

**CHANGELOG**: Usually no changelog entry. Tests alone do not change shipped behavior.

---

## Step 15 — CI & Release

**Goal**: GitHub Actions CI runs `make verify` on Linux + macOS. GoReleaser builds releases.

**Files to create**:
- `.github/workflows/ci.yml` — matrix: [ubuntu-latest, macos-latest] × go 1.24, runs `make verify`
- `.github/workflows/release.yml` — triggered on tag push, runs GoReleaser

**CI workflow** (`.github/workflows/ci.yml`):
```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  verify:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go install honnef.co/go/tools/cmd/staticcheck@latest
      - run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: make verify
```

**Validation**:
```bash
# Local: make verify passes
# Push branch, open PR → CI runs on both OS targets
# Tag v0.1.0 → GoReleaser produces binaries
```

**CHANGELOG**: Usually no changelog entry. CI/release automation changes are not recorded unless they alter shipped product behavior.

---

## Dependency Graph

```
Step 0  (scaffold)
  └─ Step 1  (cobra CLI)
       └─ Step 2  (catalog)
            ├─ Step 3  (pkgmgr)
            ├─ Step 4  (discovery)
            ├─ Step 5  (execx)
            │    └─ Step 6  (verify)
            ├─ Step 7  (state)
            ├─ Step 8  (plan) ← needs catalog + pkgmgr + discovery + state
            │    └─ Step 9  (plan execution) ← needs verify + state
            ├─ Step 10 (skill) ← needs catalog + state/discovery outputs
            ├─ Step 11 (shell) ← needs catalog + state
            └─ Step 12 (tui) ← needs catalog + discovery
  └─ Step 13 (wire commands) ← needs ALL of 1-12
  └─ Step 14 (integration tests) ← needs 13
  └─ Step 15 (CI/release) ← needs 0
```

**Parallelizable after Step 2**: Steps 3, 5, 7, 10, 11, 12 can proceed in parallel once catalog exists. Step 4 should land before wiring commands because it defines the runtime snapshot. Step 6 needs 5. Step 8 needs 3+4+7. Step 9 needs 6+7+8.

---

## Rules for Every Step

1. **Run `make verify` after every step** — no exceptions
2. **Update `CHANGELOG.md` only for functional changes** under `## [Unreleased]`
3. Creating the initial `CHANGELOG.md` file is part of Step 0, but scaffolding, refactors, test-only changes, CI-only changes, and internal docs-only changes do not get release notes unless they change shipped behavior
4. **No `sh -c`** — all external commands via `os/exec` directly
5. **Interfaces at boundaries** — executor, manager, verifier are all interfaces for testability
6. **No business logic in cobra handlers or bubbletea views**
7. **Every new package must have tests** wired into `go test ./...`
8. **Commit after each step passes verification**
