package cmd

import (
	"github.com/enjoypi/bkpic/cmd/internal/tidy"
	"github.com/spf13/cobra"
)

// doCmd represents the do command
var tidyCmd = &cobra.Command{
	Use:     "tidy",
	Short:   "tidy media to output directory",
	PreRunE: preRunE,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tidy.Run(rootViper, args)
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(tidyCmd)

}
