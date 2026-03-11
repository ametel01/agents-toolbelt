package main

import "github.com/spf13/cobra"

func newInstallCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install selected tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = yes
			cmd.Println("not implemented")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Install default tools without prompting")

	return cmd
}
