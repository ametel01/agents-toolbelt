package main

import "github.com/spf13/cobra"

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [tool]",
		Short: "Update atb-managed tools",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("not implemented")

			return nil
		},
	}
}
