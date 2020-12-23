package api

import (
	"github.com/creekorful/trandoshan/internal/api/auth"
	"github.com/creekorful/trandoshan/internal/api/rest"
	"github.com/creekorful/trandoshan/internal/api/service"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// GetApp return the api app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-api",
		Version: "0.7.0",
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
			&cli.StringFlag{
				Name:  "refresh-delay",
				Usage: "Duration before allowing indexation of existing resource (none = never)",
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
	svc, err := service.New(c)
	if err != nil {
		log.Err(err).Msg("error while creating API service")
		return err
	}

	// Setup middlewares
	authMiddleware := auth.NewMiddleware(signingKey)
	e.Use(authMiddleware.Middleware())

	// Add endpoints
	e.GET("/v1/resources", rest.SearchResources(svc))
	e.POST("/v1/resources", rest.AddResource(svc))
	e.POST("/v1/urls", rest.ScheduleURL(svc))

	log.Info().Msg("Successfully initialized tdsh-api. Waiting for requests")

	return e.Start(":8080")
}
