package feeder

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/util/logging"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// GetApp return the feeder app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-feeder",
		Version: "0.3.0",
		Usage:   "Trandoshan feeder process",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			&cli.StringFlag{
				Name:     "api-uri",
				Usage:    "URI to the API server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "url",
				Usage:    "URL to send to the crawler",
				Required: true,
			},
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)

	log.Info().Str("ver", ctx.App.Version).Msg("Starting tdsh-feeder")

	log.Debug().Str("uri", ctx.String("api-uri")).Msg("Using API server")

	apiClient := api.NewClient(ctx.String("api-uri"))
	if err := apiClient.AddURL(ctx.String("url")); err != nil {
		log.Err(err).Msg("Unable to publish URL")
		return err
	}

	log.Info().Str("url", ctx.String("url")).Msg("URL successfully sent to the crawler")

	return nil
}
