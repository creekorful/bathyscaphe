package util

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/urfave/cli/v2"
)

// GetAPITokenFlag return the cli flag to provide API token
func GetAPITokenFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "api-token",
		Usage:    "Token to use to authenticate against the API",
		Required: true,
	}
}

// GetAPIURIFlag return the cli flag to set api uri
func GetAPIURIFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "api-uri",
		Usage:    "URI to the API server",
		Required: true,
	}
}

// GetAPIClient return a new configured API client
func GetAPIClient(c *cli.Context) api.Client {
	return api.NewClient(c.String("api-uri"), c.String("api-token"))
}
