package feeder

import (
	"bytes"
	"fmt"
	"github.com/creekorful/trandoshan/internal/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"net/http"
)

// GetApp return the feeder app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-feeder",
		Version: "0.0.1",
		Usage:   "Trandoshan feeder process",
		Flags: []cli.Flag{
			log.GetLogFlag(),
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
	log.ConfigureLogger(ctx)

	logrus.Infof("Starting trandoshan-feeder v%s", ctx.App.Version)

	logrus.Debugf("Using API server at: %s", ctx.String("api-uri"))

	apiURL := fmt.Sprintf("%s/v1/urls", ctx.String("api-uri"))
	if _, err := http.Post(apiURL, "application/json", bytes.NewBufferString(ctx.String("api-uri"))); err != nil {
		logrus.Errorf("Unable to publish URL: %s", err)
		return err
	}

	logrus.Infof("URL %s successfully sent to the crawler", ctx.String("url"))

	return nil
}
