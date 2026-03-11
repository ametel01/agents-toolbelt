package main

import "github.com/spf13/cobra"

func newInstallCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install selected tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInstall(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), yes)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Install default tools without prompting")

	return cmd
}
