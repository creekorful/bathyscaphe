package feeder

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/util/http"
	"github.com/creekorful/trandoshan/internal/util/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// GetApp return the feeder app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-feeder",
		Version: "0.1.0",
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

	logrus.Infof("Starting tdsh-feeder v%s", ctx.App.Version)

	logrus.Debugf("Using API server at: %s", ctx.String("api-uri"))

	apiURL := fmt.Sprintf("%s/v1/urls", ctx.String("api-uri"))

	c := http.Client{}
	res, err := c.JSONPost(apiURL, ctx.String("url"), nil)
	if err != nil {
		logrus.Errorf("Unable to publish URL: %s", err)
		if res != nil {
			logrus.Errorf("Received status code: %d", res.StatusCode)
		}
		return err
	}

	logrus.Infof("URL %s successfully sent to the crawler", ctx.String("url"))

	return nil
}
