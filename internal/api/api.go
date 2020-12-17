package api

import (
	"github.com/creekorful/trandoshan/internal/api/auth"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/labstack/echo/v4"
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
		Version: "0.5.1",
		Usage:   "Trandoshan API component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetHubURI(),
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
			&cli.StringSliceFlag{
				Name:     "users",
				Usage:    "List of API users. (Format user:password)",
				Required: false,
			},
		},
		Action: execute,
	}
}

func execute(c *cli.Context) error {
	logging.ConfigureLogger(c)

	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		log.Err(err).Msg("error while processing API call")
		e.DefaultHTTPErrorHandler(err, c)
	}
	e.HideBanner = true

	log.Info().Str("ver", c.App.Version).
		Str("elasticsearch-uri", c.String("elasticsearch-uri")).
		Str("hub-uri", c.String("hub-uri")).
		Msg("Starting tdsh-api")

	signingKey := []byte(c.String("signing-key"))

	// Create the service
	svc, err := newService(c)
	if err != nil {
		log.Err(err).Msg("Unable to start API")
		return err
	}

	// Setup middlewares
	authMiddleware := auth.NewMiddleware(signingKey)
	e.Use(authMiddleware.Middleware())

	// Add endpoints
	e.GET("/v1/resources", searchResourcesEndpoint(svc))
	e.POST("/v1/resources", addResourceEndpoint(svc))
	e.POST("/v1/urls", scheduleURLEndpoint(svc))

	log.Info().Msg("Successfully initialized tdsh-api. Waiting for requests")

	return e.Start(":8080")
}
