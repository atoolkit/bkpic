package main

import (
	"bytes"
	"github.com/enjoypi/bkpic/cmd"
	"github.com/sirupsen/logrus"
	"strings"
)

func main() {
	logrus.SetFormatter(&formatter{})
	cmd.Execute()
}

type formatter struct {
}

func (f *formatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := bytes.NewBufferString(strings.ToUpper(entry.Level.String()))
	b.WriteString("\t")
	b.WriteString(entry.Message)
	b.WriteString("\n")
	return b.Bytes(), nil
}
