package main

import "github.com/spf13/cobra"

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [tool]",
		Short: "Update atb-managed tools",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var toolID string
			if len(args) == 1 {
				toolID = args[0]
			}

			return runUpdate(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), toolID)
		},
	}
}
