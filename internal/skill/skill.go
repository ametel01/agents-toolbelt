// Package skill generates the cli-tools skill file for coding agents.
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

const skillHeader = `---
name: cli-tools
description: >-
  Use when working with CLI tools in the terminal. Lists verified CLI tools
  available on this system, grouped by category. Activate when a task involves
  terminal commands, file operations, API calls, or development workflows.
---

# CLI Tools Inventory

## Available Tools
`

var categoryOrder = []string{
	"Navigation",
	"File Viewing",
	"Filesystem",
	"Diffing",
	"JSON / YAML",
	"HTTP / APIs",
	"Environment",
	"Task Running",
	"Benchmarking",
	"Runtime Management",
	"Cloud",
	"Kubernetes",
	"Infrastructure as Code",
	"Terminal UIs",
}

var categoryLabels = map[string]string{
	"benchmarking":    "Benchmarking",
	"cloud_gcp":       "Cloud",
	"diff_viewing":    "Diffing",
	"docker_tui":      "Terminal UIs",
	"env_management":  "Environment",
	"file_viewing":    "File Viewing",
	"filesystem":      "Filesystem",
	"forge":           "HTTP / APIs",
	"grpc_api":        "HTTP / APIs",
	"http_api":        "HTTP / APIs",
	"iac":             "Infrastructure as Code",
	"json":            "JSON / YAML",
	"kubernetes":      "Kubernetes",
	"kubernetes_tui":  "Terminal UIs",
	"navigation":      "Navigation",
	"python_runtime":  "Runtime Management",
	"runtime_manager": "Runtime Management",
	"shell_history":   "Environment",
	"shell_prompt":    "Environment",
	"task_runner":     "Task Running",
	"yaml":            "JSON / YAML",
}

// Generate renders the cli-tools skill content for verified, exposable tools.
func Generate(tools []catalog.Tool) string {
	grouped := make(map[string][]string)

	for _, tool := range tools {
		if !tool.SkillExpose {
			continue
		}

		category := skillCategory(tool.Category)
		grouped[category] = append(grouped[category], tool.Bin)
	}

	var builder strings.Builder
	builder.WriteString(skillHeader)

	if len(grouped) == 0 {
		builder.WriteString("\nNo verified tools available.\n")

		return builder.String()
	}

	for _, category := range orderedCategories(grouped) {
		bins := grouped[category]
		slices.Sort(bins)
		builder.WriteString("\n### ")
		builder.WriteString(category)
		builder.WriteString("\n")

		for _, bin := range bins {
			builder.WriteString("- `")
			builder.WriteString(bin)
			builder.WriteString("`\n")
		}
	}

	return builder.String()
}

// Write persists generated skill content to all requested output paths.
func Write(content string, paths []string) error {
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return fmt.Errorf("create skill directory for %s: %w", path, err)
		}

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return fmt.Errorf("write skill file %s: %w", path, err)
		}
	}

	return nil
}

// DefaultPaths returns the standard cli-tools skill destinations for Claude Code and Codex.
func DefaultPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	return []string{
		filepath.Join(homeDir, ".claude", "skills", "cli-tools", "SKILL.md"),
		filepath.Join(homeDir, ".agents", "skills", "cli-tools", "SKILL.md"),
	}
}

func orderedCategories(grouped map[string][]string) []string {
	ordered := make([]string, 0, len(grouped))
	seen := make(map[string]struct{}, len(grouped))

	for _, category := range categoryOrder {
		if _, ok := grouped[category]; ok {
			ordered = append(ordered, category)
			seen[category] = struct{}{}
		}
	}

	extras := make([]string, 0, len(grouped))
	for category := range grouped {
		if _, ok := seen[category]; !ok {
			extras = append(extras, category)
		}
	}

	slices.Sort(extras)

	return append(ordered, extras...)
}

func skillCategory(category string) string {
	if label, ok := categoryLabels[category]; ok {
		return label
	}

	return category
}
