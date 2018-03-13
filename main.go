package main

import (
	"bytes"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&formatter{})
}

func main() {
	app := cli.NewApp()
	app.Name = "bkpic"
	app.Usage = "arrange media"
	app.Action = rootAction
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "dry-run, n", Usage: "perform a trial run with no changes made"},
		cli.BoolFlag{Name: "move, m", Usage: "move"},
		cli.BoolFlag{Name: "verbose, V", Usage: "verbose"},
	}

	app.Run(os.Args)
}

type formatter struct {
}

func (f *formatter) Format(entry *log.Entry) ([]byte, error) {
	b := bytes.NewBufferString(strings.ToUpper(entry.Level.String()))
	b.WriteString("\t")
	b.WriteString(entry.Message)
	b.WriteString("\n")
	return b.Bytes(), nil
}
