package log

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// GetLogFlag return the CLI flag parameter used to setup application log level
func GetLogFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:  "log-level",
		Usage: "Set the application log level",
		Value: "info",
	}
}

// ConfigureLogger configure the logger using given log level (read from cli context)
func ConfigureLogger(ctx *cli.Context) {
	// Set application log level
	if lvl, err := logrus.ParseLevel(ctx.String("log-level")); err == nil {
		logrus.SetLevel(lvl)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}
