package util

import "github.com/urfave/cli/v2"

// GetEventSrvURI return the URI of the event server
func GetEventSrvURI() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "event-srv-uri",
		Usage:    "URI to the event server",
		Required: true,
	}
}
