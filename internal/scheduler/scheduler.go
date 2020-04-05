package scheduler

import (
	"fmt"
	"github.com/PuerkitoBio/purell"
	"github.com/creekorful/trandoshan/internal/log"
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
			log.GetLogFlag(),
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
	log.ConfigureLogger(ctx)

	logrus.Infof("Starting trandoshan-scheduler v%s", ctx.App.Version)

	logrus.Debugf("Using NATS server at: %s", ctx.String("nats-uri"))

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	logrus.Info("Successfully initialized trandoshan-scheduler. Waiting for URLs")

	if err := sub.QueueSubscribe(proto.URLFoundSubject, "schedulers", handleMessage()); err != nil {
		return err
	}

	return nil
}

func handleMessage() natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var urlMsg proto.URLFoundMsg
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

		logrus.Debugf("Normalized URL: %s", normalizedURL)

		// TODO implement scheduling logic

		if err := natsutil.PublishJSON(nc, proto.URLTodoSubject, &proto.URLTodoMsg{URL: urlMsg.URL}); err != nil {
			return fmt.Errorf("error while publishing URL: %s", err)
		}

		return nil
	}
}
