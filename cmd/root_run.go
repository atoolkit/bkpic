package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// config for root
type cfgRoot struct {
	Etcd   string
	Server server
}

type server struct {
	ListenAddress string
}

func run(v *viper.Viper) error {
	var c cfgRoot
	if err := v.Unmarshal(&c); err != nil {
		return err
	}
	logrus.Infof("settings: %+v", c)
	return nil
}
