package util

import "github.com/urfave/cli/v2"

// GetConfigApiURIFlag return the cli flag to set config api uri
func GetConfigApiURIFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "config-api-uri",
		Usage:    "URI to the ConfigAPI server",
		Required: true,
	}
}