package api

import (
	tlog "github.com/creekorful/trandoshan/internal/log"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/urfave/cli/v2"
)

// GetApp return the api app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-api",
		Version: "0.0.1",
		Usage:   "", // TODO
		Flags: []cli.Flag{
			tlog.GetLogFlag(),
			&cli.StringFlag{
				Name:     "elasticsearch-uri",
				Usage:    "URI to the Elasticsearch server",
				Required: true,
			},
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	e := echo.New()
	e.HideBanner = true

	// Configure logger
	switch ctx.String("log-level") {
	case "debug":
		e.Logger.SetLevel(log.DEBUG)
	case "info":
		e.Logger.SetLevel(log.INFO)
	case "warn":
		e.Logger.SetLevel(log.WARN)
	case "error":
		e.Logger.SetLevel(log.ERROR)
	}

	e.Logger.Infof("Starting trandoshan-api v%s", ctx.App.Version)

	// Create Elasticsearch client
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{ctx.String("elasticsearch-uri")}})
	if err != nil {
		e.Logger.Errorf("Error while creating elasticsearch client: %s", err)
		return err
	}

	// Add endpoints
	e.GET("/resources", getResources(es))
	e.POST("/resources", addResource(es))

	e.Logger.Info("Successfully initialized trandoshan-api. Waiting for requests")

	return e.Start(":8080")
}

func getResources(es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		return nil // TODO
	}
}

func addResource(es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		return nil // TODO
	}
}
