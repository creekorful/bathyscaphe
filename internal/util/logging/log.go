package logging

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
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
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Set application log level
	if lvl, err := zerolog.ParseLevel(ctx.String("log-level")); err == nil {
		zerolog.SetGlobalLevel(lvl)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
