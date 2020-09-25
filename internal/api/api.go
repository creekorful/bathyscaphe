package api

import (
	logging2 "github.com/creekorful/trandoshan/internal/logging"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var (
	defaultPaginationSize = 50
	maxPaginationSize     = 100
)

// GetApp return the api app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-api",
		Version: "0.4.0",
		Usage:   "Trandoshan API component",
		Flags: []cli.Flag{
			logging2.GetLogFlag(),
			&cli.StringFlag{
				Name:     "nats-uri",
				Usage:    "URI to the NATS server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "elasticsearch-uri",
				Usage:    "URI to the Elasticsearch server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "signing-key",
				Usage:    "Signing key for the JWT token",
				Required: true,
			},
		},
		Action: execute,
	}
}

func execute(c *cli.Context) error {
	logging2.ConfigureLogger(c)

	e := echo.New()
	e.HideBanner = true

	log.Info().Str("ver", c.App.Version).Msg("Starting tdsh-api")

	log.Debug().Str("uri", c.String("elasticsearch-uri")).Msg("Using Elasticsearch server")
	log.Debug().Str("uri", c.String("nats-uri")).Msg("Using NATS server")

	signingKey := []byte(c.String("signing-key"))

	// Create the service
	svc, err := newService(c, signingKey)
	if err != nil {
		log.Err(err).Msg("Unable to start API")
		return err
	}

	// Setup middlewares
	e.Use(middleware.JWT(signingKey))

	// Add endpoints
	e.GET("/v1/resources", searchResourcesEndpoint(svc))
	e.POST("/v1/resources", addResourceEndpoint(svc))
	e.POST("/v1/urls", scheduleURLEndpoint(svc))
	e.POST("/v1/sessions", authenticateEndpoint(svc))

	log.Info().Msg("Successfully initialized tdsh-api. Waiting for requests")

	return e.Start(":8080")
}
