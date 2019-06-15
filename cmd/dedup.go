package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type dedupConfig struct {
	DryRun bool `mapstructure:"dry-run"`
}

var files = make(map[string][]string) // {hashValue, fileSlice}

// doCmd represents the do command
var dedupCmd = &cobra.Command{
	Use:     "dedup",
	Short:   "Deduplication files",
	PreRunE: preRunE,
	RunE: func(cmd *cobra.Command, args []string) error {
		return dedup(rootViper, args)
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(dedupCmd)
	flags := dedupCmd.Flags()
	flags.BoolP("dry-run", "n", false, "perform a trial run with no changes made")
}

func dedup(v *viper.Viper, args []string) error {
	var c dedupConfig
	if err := v.Unmarshal(&c); err != nil {
		return err
	}

	logrus.Infof("settings on dedup: %+v", c)
	logrus.Info("args: ", args)
	return doDedup(&c, args)
}

func doDedup(c *dedupConfig, args []string) error {
	dir, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	info, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not directory", args[0])
	}

	if err := filepath.Walk(dir, walk); err != nil {
		return err
	}

	for _, f := range files {
		if len(f) <= 1 {
			continue
		}

		for _, path := range f {
			fmt.Println(path)
		}
		fmt.Println()
	}
	return nil
}

func walk(path string, info os.FileInfo, err error) error {

	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	strHash, err := hashStr(path, sha256.New())
	if err != nil {
		return err
	}

	logrus.Trace(path)
	_, found := files[strHash]
	if found {
	}

	files[strHash] = append(files[strHash], path)
	if found {
		logrus.Debug(files[strHash])
	}
	return nil
}

func hashStr(filename string, h hash.Hash) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
