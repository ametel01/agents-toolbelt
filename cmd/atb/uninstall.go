package main

import (
	"errors"

	"github.com/spf13/cobra"
)

var errUninstallTargetRequired = errors.New("specify a tool name or use --all")

func newUninstallCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "uninstall [tool]",
		Short: "Uninstall atb-managed tools",
		Args: func(_ *cobra.Command, args []string) error {
			if all && len(args) == 0 {
				return nil
			}

			if !all && len(args) == 1 {
				return nil
			}

			return errUninstallTargetRequired
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args, all)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Uninstall all atb-managed tools")

	return cmd
}
