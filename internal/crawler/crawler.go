package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"mvdan.cc/xurls/v2"
)

const (
	TodoSubject = "todo"
	DoneSubject = "done"
)

// UrlMessage represent an URL as read by the crawler process
type UrlMessage struct {
	Url string `json:"url"`
}

func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-crawler",
		Version: "0.0.1",
		Usage:   "", // TODO
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "nats-uri",
				Usage:    "URI to the NATS server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "tor-uri",
				Usage:    "URI to the TOR SOCKS proxy",
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

	logrus.Infof("Starting trandoshan-crawler v%s", ctx.App.Version)

	logrus.Debugf("Using NATS server at: %s", ctx.String("nats-uri"))
	logrus.Debugf("Using TOR proxy at: %s", ctx.String("tor-uri"))

	// Connect to the NATS server
	nc, err := nats.Connect(ctx.String("nats-uri"))
	if err != nil {
		logrus.Errorf("Error while connecting to NATS server %s: %s", ctx.String("nats-uri"), err)
		return err
	}
	defer nc.Close()

	// Create the subscriber
	sub, err := nc.QueueSubscribeSync(TodoSubject, "crawlers")
	if err != nil {
		logrus.Errorf("Error while reading message from NATS server: %s", err)
		return err
	}

	logrus.Info("Successfully initialized trandoshan-crawler. Waiting for URLs")

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
	var urlMsg UrlMessage
	if err := json.Unmarshal(msg.Data, &urlMsg); err != nil {
		return fmt.Errorf("error while decoding message: %s", err)
	}

	logrus.Debugf("Processing URL: %s", urlMsg.Url)

	httpClient := fasthttp.Client{}
	_, body, err := httpClient.Get(nil, urlMsg.Url)
	if err != nil {
		return err
	}

	// Extract URLs
	xu := xurls.Strict()
	urls := xu.FindAllString(string(body), -1)

	// Publish found URLs
	for _, url := range urls {
		logrus.Debugf("Found URL: %s", url)

		if err := natsutil.PublishJson(nc, DoneSubject, &UrlMessage{Url: url}); err != nil {
			logrus.Warnf("Error while publishing URL: %s", err)
		}
	}

	return nil // TODO
}