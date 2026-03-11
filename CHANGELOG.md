# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-03-11

### Added
- Initial Go project scaffold, build verification workflow, and release configuration for `atb`.
- Cobra-based CLI skeleton with `install`, `status`, `update`, `uninstall`, and `catalog` subcommands.
- Embedded 22-tool registry with validation and lookup helpers for catalog-driven workflows.
- Platform normalization and package-manager detection with selection logic for catalog install methods.
- PATH discovery and reconciliation primitives that classify catalog tools as managed, external, or missing.
- Safe external command execution with timeout, output capture, and preserved exit codes for non-zero results.
- Tool verification flow that checks PATH presence, runs verify commands, and extracts versions when available.
- Persistent state management for ownership receipts, verification metadata, and atomic config writes.
- Install, update, and uninstall planning for managed versus external tools with tier-aware ordering.
- Plan execution that installs, updates, verifies, and uninstalls tools while keeping receipts and summaries consistent.
- CLI skill generation for Claude Code and Codex based on verified, exposable tool inventory.
- Shell hook suggestion and idempotent rc-file application support for tools that need initialization.
- Interactive Bubble Tea picker with preselection, search, collapse/expand behavior, and installed markers.
- End-to-end command wiring for `install`, `status`, `update`, `uninstall`, and `catalog`.
- Integration coverage for full lifecycle flows, realistic skill output, and state idempotency.
- GitHub Actions CI and GoReleaser release workflows for verification and tagged builds.

### Changed
- Installer guidance now presents `must` tools as the recommended baseline instead of preselecting them automatically.
- The standalone installer now defaults to `~/.local/bin` for non-root installs and only targets `/usr/local/bin` when run as `root`, avoiding implicit privilege escalation.
- The Bubble Tea picker now opens in the alternate screen, keeps the cursor visible in smaller terminals, shows tool descriptions inline, and merges related categories under friendlier headings.
- Picker search now also matches humanized category names and tool descriptions for easier discovery.
- Registry metadata and README copy now use more task-oriented tool descriptions and updated language for the recommended baseline flow.
- `install`, `update`, and `uninstall` now print detected package managers, plan previews, per-step progress, verification status, and skill generation destinations as they run.
- Category label mapping is now defined once in the `catalog` package and shared by skill generation and the TUI picker.
- `skill.DefaultPaths` now returns an error instead of silently returning nil when the home directory cannot be resolved.
- Shell rc-file writes now use atomic temp-file-and-rename to prevent truncation on interrupted writes.
- Update and uninstall plans now sort deterministically by tool ID within the same tier.
- `State.LastRunAt` is now set automatically on every state save.

### Fixed
- Runtime command error handling now wraps writer, verifier, and package-manager failures consistently, and the install flow has been refactored to keep linted control flow within limits.
- The standalone installer now emits concrete `resolution:` hints for missing prerequisites, unsupported platforms, download and extraction failures, and unwritable install directories.
- The standalone installer cleanup path no longer references an out-of-scope tempdir on exit.
- The interactive picker now clears selections when the user quits with `q` or `esc`, preventing unintended tool installations.
- The `uninstall` command's `persistVerifiedSkill` error is now wrapped consistently with the rest of the runtime error handling.

### Removed
- Unused `internal/logx` package.
