package feeder

import (
	"encoding/json"
	"github.com/creekorful/trandoshan-crawler/internal/crawler"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-feeder",
		Version: "0.0.1",
		Usage:   "", // TODO
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "nats-uri",
				Usage:    "URI to the NATS server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "url",
				Usage:    "URL to send to the crawler",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Set the application log level",
				Value: "info",
			},
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	// Set application log level
	if lvl, err := logrus.ParseLevel(ctx.String("log-level")); err == nil {
		logrus.SetLevel(lvl)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.Infof("Starting trandoshan-feeder v%s", ctx.App.Version)

	logrus.Debugf("Using NATS server at: %s", ctx.String("nats-uri"))

	// Connect to the NATS server
	nc, err := nats.Connect(ctx.String("nats-uri"))
	if err != nil {
		logrus.Errorf("Error while connecting to NATS server %s: %s", ctx.String("nats-uri"), err)
		return err
	}
	defer nc.Close()

	// Marshal the message
	msg, err := json.Marshal(&crawler.UrlMessage{Url: ctx.String("url")})
	if err != nil {
		logrus.Errorf("Unable to marshal message: %s", err)
		return err
	}

	// Publish the message
	if err := nc.Publish(crawler.TodoSubject, msg); err != nil {
		logrus.Errorf("Unable to publish message: %s", err)
		return err
	}

	logrus.Infof("Url %s successfully sent to the crawler", ctx.String("url"))

	return nil
}
