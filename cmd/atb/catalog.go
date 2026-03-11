package main

import "github.com/spf13/cobra"

func newCatalogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "catalog",
		Short: "List tools available in the catalog",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("not implemented")

			return nil
		},
	}
}
