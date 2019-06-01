package cmd

import (
	"github.com/enjoypi/bkpic/cmd/internal"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// doCmd represents the do command
var tidyCmd = &cobra.Command{
	Use:   "tidy",
	Short: "tidy media to output directory",
	PreRunE: preRunE,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tidy(rootViper, args)
	},
	Args:cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(tidyCmd)
	flags := tidyCmd.Flags()
	flags.BoolP("debug", "d", false, "debug")
	flags.BoolP("dry-run", "n", false, "perform a trial run with no changes made")
	flags.BoolP("move", "m", false, "move")
	flags.StringP("output", "o", "", "the output directory")

	_ = tidyCmd.MarkFlagRequired("output")
}


func tidy(v *viper.Viper, args [] string) error {
	var c internal.TidyConfig
	if err := v.Unmarshal(&c); err != nil {
		return err
	}

	logrus.Infof("settings on child: %+v", c)
	logrus.Info("args: ", args)
	return internal.Tidy(&c, args)
}
