package crawler

import (
	"context"
	"crypto/tls"
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"mvdan.cc/xurls/v2"
	"time"
)

// GetApp return the crawler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-crawler",
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
				Name:     "tor-uri",
				Usage:    "URI to the TOR SOCKS proxy",
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
	sub, err := nc.QueueSubscribeSync(proto.URLTodoSubject, "crawlers")
	if err != nil {
		logrus.Errorf("Error while reading message from NATS server: %s", err)
		return err
	}

	// Create the HTTP client
	httpClient := &fasthttp.Client{
		// Use given TOR proxy to reach the hidden services
		Dial:         fasthttpproxy.FasthttpSocksDialer(ctx.String("tor-uri")),
		// Disable SSL verification since we do not really care about this
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
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
		if err := handleMessage(nc, httpClient, msg); err != nil {
			logrus.Warnf("Skipping current message because of error: %s", err)
			continue
		}
	}

	return nil
}

func handleMessage(nc *nats.Conn, httpClient *fasthttp.Client, msg *nats.Msg) error {
	var urlMsg proto.URLTodoMessage
	if err := natsutil.ReadJSON(msg, &urlMsg); err != nil {
		return err
	}

	logrus.Debugf("Processing URL: %s", urlMsg.URL)

	// Query the website
	_, body, err := httpClient.Get(nil, urlMsg.URL)
	if err != nil {
		return err
	}

	// Extract URLs
	xu := xurls.Strict()
	urls := xu.FindAllString(string(body), -1)

	// Publish found URLs
	for _, url := range urls {
		logrus.Debugf("Found URL: %s", url)

		if err := natsutil.PublishJSON(nc, proto.URLDoneSubject, &proto.URLDoneMessage{URL: url}); err != nil {
			logrus.Warnf("Error while publishing URL: %s", err)
		}
	}

	return nil // TODO
}