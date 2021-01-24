package cmd

import (
	"github.com/enjoypi/bkpic/cmd/internal"
	"github.com/spf13/cobra"
)

type rmdupConfig struct {
	DryRun bool `mapstructure:"dry-run"`
}

var files = make(map[string][]string) // {hashValue, fileSlice}

// doCmd represents the do command
var rmdupCmd = &cobra.Command{
	Use:     "rmdup",
	Short:   "remove duplication files",
	PreRunE: preRunE,
	RunE: func(cmd *cobra.Command, args []string) error {
		return internal.Rmdup(rootViper, args)
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(rmdupCmd)
	flags := rmdupCmd.Flags()
	flags.BoolP("dry-run", "n", false, "perform a trial run with no changes made")
}
