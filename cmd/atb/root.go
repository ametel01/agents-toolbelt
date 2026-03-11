package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "atb",
	Short:        "Install and manage CLI tools for coding agent workflows",
	Version:      version,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newUpdateCmd())
	rootCmd.AddCommand(newUninstallCmd())
	rootCmd.AddCommand(newCatalogCmd())
}

// Execute runs the root command tree.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("execute root command: %w", err)
	}

	return nil
}
