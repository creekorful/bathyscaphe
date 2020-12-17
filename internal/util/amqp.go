package util

import "github.com/urfave/cli/v2"

// GetHubURI return the URI of the hub (event) server
func GetHubURI() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "hub-uri",
		Usage:    "URI to the hub (event) server",
		Required: true,
	}
}
