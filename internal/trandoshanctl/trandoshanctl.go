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
			{
				Name:      "search",
				Usage:     "Search for specific resources",
				ArgsUsage: "keyword",
				Action:    search,
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

func search(c *cli.Context) error {
	keyword := c.Args().First()
	apiClient := api.NewClient(c.String("api-uri"))

	res, err := apiClient.SearchResources("", keyword)
	if err != nil {
		log.Err(err).Str("keyword", keyword).Msg("Unable to search resources")
		return err
	}

	if len(res) == 0 {
		fmt.Println("No resources crawled (yet).")
	}

	for _, r := range res {
		fmt.Printf("%s - %s\n", r.URL, r.Title)
	}

	return nil
}
