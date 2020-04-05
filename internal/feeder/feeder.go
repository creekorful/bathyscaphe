package feeder

import (
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// GetApp return the feeder app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-feeder",
		Version: "0.0.1",
		Usage:   "", // TODO
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Set the application log level",
				Value: "info",
			},
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

	// Publish the message
	if err := natsutil.PublishJSON(nc, proto.URLTodoSubject, &proto.URLTodoMsg{URL: ctx.String("url")}); err != nil {
		logrus.Errorf("Unable to publish URL: %s", err)
		return err
	}

	logrus.Infof("URL %s successfully sent to the crawler", ctx.String("url"))

	return nil
}
