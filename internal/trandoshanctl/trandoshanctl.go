package trandoshanctl

import (
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/util/logging"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// GetApp returns the Trandoshan CLI app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshanctl",
		Version: "0.3.0",
		Usage:   "Trandoshan CLI",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			&cli.StringFlag{
				Name:  "api-uri",
				Usage: "URI to the API server",
				Value: "http://localhost:15005",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "schedule",
				Usage:     "Schedule crawling for given URL",
				Action:    schedule,
				ArgsUsage: "URL",
			},
		},
		Before: before,
	}
}

func before(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)
	return nil
}

func schedule(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("missing argument URL")
	}

	url := c.Args().First()
	apiClient := api.NewClient(c.String("api-uri"))

	if err := apiClient.ScheduleURL(url); err != nil {
		log.Err(err).Str("url", url).Msg("Unable to schedule crawling for URL")
		return err
	}

	log.Info().Str("url", url).Msg("Successfully schedule crawling")

	return nil
}
