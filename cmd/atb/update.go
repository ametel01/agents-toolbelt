package main

import (
	"context"
	"io"

	"github.com/spf13/cobra"
)

type (
	selfUpdateRunner func(context.Context, io.Writer, io.Writer) error
	toolUpdateRunner func(context.Context, io.Writer, io.Writer, string) error
)

func newUpdateCmd() *cobra.Command {
	return newUpdateCmdWithRunners(runSelfUpdate, runToolUpdate)
}

func newUpdateCmdWithRunners(selfUpdate selfUpdateRunner, toolUpdate toolUpdateRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the atb CLI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return selfUpdate(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.AddCommand(newUpdateToolsCmd(toolUpdate))

	return cmd
}

func newUpdateToolsCmd(toolUpdate toolUpdateRunner) *cobra.Command {
	return &cobra.Command{
		Use:   "tools [tool]",
		Short: "Update atb-managed tools",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var toolID string
			if len(args) == 1 {
				toolID = args[0]
			}

			return toolUpdate(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), toolID)
		},
	}
}
