package util

import "github.com/urfave/cli/v2"

// GetNATSURIFlag return the nats uri from cli flag
func GetNATSURIFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "nats-uri",
		Usage:    "URI to the NATS server",
		Required: true,
	}
}
