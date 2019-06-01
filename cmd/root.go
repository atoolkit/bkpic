package cmd

import (
	"bytes"
	"context"
	"time"

	"gopkg.in/yaml.v2"
	"github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile           string
	configRemoteEndpoint string
	configRemotePath     string
	configType           string
	logLevel             string
	rootViper            = viper.New()
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
		logrus.Fatal(err)
	}
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)


	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVarP(&configFile, "config.file", "c", "", "config file")

	rootCmd.PersistentFlags().StringVar(&configRemoteEndpoint,
		"config.remote.endpoint",
		"", //"127.0.0.1:2379",
		"the endpoint of remote config")
	rootCmd.PersistentFlags().StringVar(&configRemotePath,
		"config.remote.path",
		"bkpic/config",
		"the path of remote config")

	rootCmd.PersistentFlags().StringVar(&configType, "config.type", "yaml", "the type of config format")
	rootCmd.PersistentFlags().BoolP("verbose", "V", false, "verbose")

	rootCmd.PersistentFlags().StringVar(&logLevel, "log.level", "info", "level of logrus")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().Bool("version", false, "show version")
}

func preRunE(cmd *cobra.Command, args []string) error {

	// use flag log.level
	if lvl, err := logrus.ParseLevel(logLevel); err == nil {
		logrus.SetLevel(lvl)
		if lvl == logrus.TraceLevel {
			logrus.SetReportCaller(true)
		}
	}

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

	// remote config, key/value store
	initRemoteConfig(v)

	// local config
	if configFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(configFile)

		// If a config file is found, read it in.
		if err := v.ReadInConfig(); err != nil {
			return err
		}
		logrus.Debug("using config file: ", v.ConfigFileUsed())
		logrus.Debug("local settings: ", v.AllSettings())
	}

	// env
	v.AutomaticEnv() // read in environment variables that match

	// flag
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	// log level in flags maybe wrong, reset
	if lvl, err := logrus.ParseLevel(v.GetString("log.level")); err == nil {
		logrus.SetLevel(lvl)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
		logrus.Warn(err)
	}

	logrus.Warn("current log level: ", logrus.GetLevel())

	showConfig(v)
	return nil
}

func initRemoteConfig(v *viper.Viper) {
	if configRemoteEndpoint == "" {
		return
	}

	logrus.Info("reading from ", configRemoteEndpoint)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{configRemoteEndpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logrus.Warn(err)
		return
	}
	defer cli.Close()

	resp, err := cli.Get(context.Background(), configRemotePath)
	if err != nil {
		logrus.Warn(err)
		return
	}

	for _, kv := range resp.Kvs {
		if err := v.MergeConfig(bytes.NewBuffer(kv.Value)); err == nil {
			logrus.Debug("remote settings: ", v.AllSettings())
		} else {
			logrus.Warn(err)
		}
	}
}

func showConfig(v *viper.Viper) {
	if out, err := yaml.Marshal(v.AllSettings()); err == nil {
		logrus.Debug("all settings:\n", string(out))
	} else {
		logrus.Debug("all settings: ", v.AllSettings())
	}
}
