package scheduler

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/purell"
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// GetApp return the scheduler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-scheduler",
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

	logrus.Infof("Starting trandoshan-scheduler v%s", ctx.App.Version)

	logrus.Debugf("Using NATS server at: %s", ctx.String("nats-uri"))

	// Connect to the NATS server
	nc, err := nats.Connect(ctx.String("nats-uri"))
	if err != nil {
		logrus.Errorf("Error while connecting to NATS server %s: %s", ctx.String("nats-uri"), err)
		return err
	}
	defer nc.Close()

	// Create the subscriber
	sub, err := nc.QueueSubscribeSync(proto.URLDoneSubject, "schedulers")
	if err != nil {
		logrus.Errorf("Error while reading message from NATS server: %s", err)
		return err
	}

	logrus.Info("Successfully initialized trandoshan-scheduler. Waiting for URLs")

	for {
		// Read incoming message
		msg, err := sub.NextMsgWithContext(context.Background())
		if err != nil {
			logrus.Warnf("Skipping current message because of error: %s", err)
			continue
		}

		// ... And process it
		if err := handleMessage(nc, msg); err != nil {
			logrus.Warnf("Skipping current message because of error: %s", err)
			continue
		}
	}

	return nil
}

func handleMessage(nc *nats.Conn, msg *nats.Msg) error {
	var urlMsg proto.URLDoneMessage
	if err := natsutil.ReadJSON(msg, &urlMsg); err != nil {
		return err
	}

	logrus.Debugf("Processing URL: %s", urlMsg.URL)

	// Normalized received URL
	normalizedURL, err := purell.NormalizeURLString(urlMsg.URL, purell.FlagsUsuallySafeGreedy|
		purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)

	if err != nil {
		return fmt.Errorf("error while normalizing URL %s: %s", urlMsg.URL, err)
	}

	logrus.Debugf("Normalizing URL: %s", normalizedURL)

	// TODO implement scheduling logic

	if err := natsutil.PublishJSON(nc, proto.URLTodoSubject, &proto.URLTodoMessage{URL: urlMsg.URL}); err != nil {
		return fmt.Errorf("error while publishing URL: %s", err)
	}

	return nil
}
