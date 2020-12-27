package util

import "github.com/urfave/cli/v2"

// GetConfigAPIURIFlag return the cli flag to set config api uri
func GetConfigAPIURIFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "config-api-uri",
		Usage:    "URI to the ConfigAPI server",
		Required: true,
	}
}
