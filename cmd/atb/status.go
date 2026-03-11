package main

import "github.com/spf13/cobra"

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show tool installation status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("not implemented")

			return nil
		},
	}
}
