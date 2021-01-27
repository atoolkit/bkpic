package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var (
	configFile string
	configType string
	logLevel   string
	rootViper  = viper.New()
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bkpic",
	Short: "backup media files",

	PreRunE: preRunE,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(rootViper)
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		zap.L().Fatal(err.Error())
	}
}

func init() {

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVarP(&configFile, "config.file", "c", "", "config file")

	rootCmd.PersistentFlags().StringVar(&configType, "config.type", "yaml", "the type of config format")
	rootCmd.PersistentFlags().BoolP("verbose", "V", false, "verbose")

	rootCmd.PersistentFlags().StringVar(&logLevel, "log.level", "debug", "level of zap")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().Bool("version", false, "show version")
}

func preRunE(cmd *cobra.Command, args []string) error {

	// use flag log.level
	var logger *zap.Logger
	var err error
	if strings.ToLower(logLevel) == "debug" {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return err
	}
	zap.ReplaceGlobals(logger)

	// Viper uses the following precedence order. Each item takes precedence over the item below it:
	//
	// explicit call to Set
	// flag
	// env
	// config
	// key/value store
	// default
	//
	// Viper configuration keys are case insensitive.

	v := rootViper
	v.SetConfigType(configType)

	// local config
	if configFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(configFile)

		// If a config file is found, read it in.
		if err := v.ReadInConfig(); err != nil {
			return err
		}
		zap.S().Debug("using config file: ", v.ConfigFileUsed())
		zap.S().Debug("local settings: ", v.AllSettings())
	}

	// env
	v.AutomaticEnv() // read in environment variables that match

	// flag
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	showConfig(v)
	return nil
}

func showConfig(v *viper.Viper) {
	if out, err := yaml.Marshal(v.AllSettings()); err == nil {
		zap.S().Debug("all settings:\n", string(out))
	} else {
		zap.S().Debug("all settings: ", v.AllSettings())
	}
}
