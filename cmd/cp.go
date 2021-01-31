package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	subCmd := &cobra.Command{
		Use:     "cp",
		Short:   "remove duplication files",
		PreRunE: preRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
			// return internal.cp.Run(rootViper, args)
		},
		Args: cobra.MinimumNArgs(-1),
	}

	flags := subCmd.Flags()
	flags.BoolP("move", "m", false, "move")
	flags.StringP("output", "o", "", "the output directory")

	_ = subCmd.MarkFlagRequired("output")
	rootCmd.AddCommand(subCmd)
}
