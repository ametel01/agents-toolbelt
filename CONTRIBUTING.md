# Contributing to `agents-toolbelt`

This project is a Go CLI application with strict local and CI verification.
If you change code, assume it must pass the same gates locally and in GitHub Actions before it is acceptable.

## Development setup

Requirements:

- Go 1.26 or newer
- `make`
- a Unix-like shell environment

The repository bootstraps linter and vulnerability-check binaries into `./.tools/bin` through the `Makefile`, so you do not need to install every tool globally.

## Repository workflow

1. Create a focused branch or work in a clean local branch.
2. Keep changes scoped to one concern when possible.
3. Run the full quality gate locally before asking for review.
4. Update docs when behavior, workflow, or developer expectations change.
5. Keep commits logical and readable. Prefer a small sequence of focused commits over one large mixed commit.

## Quality gates

The canonical verification command is:

```bash
make verify
```

`make verify` runs:

- `fmt`
- `vet`
- `lint`
- `test`
- `build`
- `vulncheck`

You can also run the individual targets during development:

```bash
make fmt
make vet
make lint
make test
make build
make vulncheck
```

Do not merge or submit changes that only pass a subset of these checks.

## Code standards

### General

- Write idiomatic Go and keep packages small and focused.
- Prefer thin command handlers and move behavior into internal packages.
- Add tests for every new package and for any non-trivial behavior change.
- Use explicit, contextual error wrapping at package boundaries.
- Do not bypass lints with `nolint` unless there is a concrete reason and the directive is narrowly scoped.

### Command execution and safety

- Do not execute install/update/uninstall actions through `sh -c`.
- Use structured command arguments.
- Preserve the ownership model: only `atb`-managed tools may be updated or uninstalled by `atb`.
- Do not silently mutate shell rc files without explicit user confirmation.

### State and behavior

- Treat persisted state as authoritative for ownership.
- Keep install receipts even when verification fails after installation.
- Keep externally discovered tools visible, but never treat them as `atb`-managed without a receipt.
- Prefer deterministic behavior and deterministic test data.

### Documentation

- Update `README.md` when user-facing behavior changes.
- Update `CHANGELOG.md` for functional or user-visible changes.
- Keep developer guidance in this file aligned with the actual repo workflow and toolchain.

## Lint profile

The repo uses a strict `golangci-lint` configuration in [`.golangci.yml`](/home/ametel/source/agents-toolbelt/.golangci.yml). In practice this means contributors should expect enforcement around:

- error handling and wrapping
- exported identifiers and package comments
- complexity limits
- test hygiene
- basic security checks
- unnecessary allocations, conversions, and unused code

If a change trips one of these rules, prefer improving the code over weakening the lint configuration.

## Testing expectations

- New packages must include tests.
- Bug fixes should include regression coverage where practical.
- Integration behavior should be tested with fakes or controlled inputs rather than relying on machine-specific state.
- Keep tests parallelizable unless they mutate process-global state such as environment variables.

## CI and releases

GitHub Actions runs `make verify` on Linux and macOS.
Tagged releases are built through GoReleaser via `.github/workflows/release.yml`.

If you change:

- the build pipeline: update the workflow files and verify locally
- release packaging: update `.goreleaser.yaml`
- quality gates: update the `Makefile`, this document, and any relevant CI steps together

## Pull request checklist

Before opening a PR, confirm:

- the code builds
- `make verify` passes
- tests cover the new behavior
- docs are updated if needed
- the change is split into logical commits when appropriate
