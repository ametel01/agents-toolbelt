# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial Go project scaffold, build verification workflow, and release configuration for `atb`.
- Cobra-based CLI skeleton with `install`, `status`, `update`, `uninstall`, and `catalog` subcommands.
- Embedded 22-tool registry with validation and lookup helpers for catalog-driven workflows.
- Platform normalization and package-manager detection with selection logic for catalog install methods.
- PATH discovery and reconciliation primitives that classify catalog tools as managed, external, or missing.
- Safe external command execution with timeout, output capture, and preserved exit codes for non-zero results.
- Tool verification flow that checks PATH presence, runs verify commands, and extracts versions when available.
