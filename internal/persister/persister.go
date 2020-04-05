package persister

import (
	"context"
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// GetApp return the persister app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-persister",
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

	logrus.Infof("Starting trandoshan-persister v%s", ctx.App.Version)

	logrus.Debugf("Using NATS server at: %s", ctx.String("nats-uri"))

	// Connect to the NATS server
	nc, err := nats.Connect(ctx.String("nats-uri"))
	if err != nil {
		logrus.Errorf("Error while connecting to NATS server %s: %s", ctx.String("nats-uri"), err)
		return err
	}
	defer nc.Close()

	// Create the subscriber
	sub, err := nc.QueueSubscribeSync(proto.ResourceSubject, "persisters")
	if err != nil {
		logrus.Errorf("Error while reading message from NATS server: %s", err)
		return err
	}

	logrus.Info("Successfully initialized trandoshan-persister. Waiting for resources")

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
	var resMsg proto.ResourceMsg
	if err := natsutil.ReadJSON(msg, &resMsg); err != nil {
		return err
	}

	logrus.Debugf("Processing resource: %s", resMsg.URL)

	return nil
}
