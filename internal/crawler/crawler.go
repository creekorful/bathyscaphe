package crawler

import (
	"crypto/tls"
	"github.com/creekorful/trandoshan/internal/log"
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
			log.GetLogFlag(),
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
	log.ConfigureLogger(ctx)

	logrus.Infof("Starting trandoshan-crawler v%s", ctx.App.Version)

	logrus.Debugf("Using NATS server at: %s", ctx.String("nats-uri"))
	logrus.Debugf("Using TOR proxy at: %s", ctx.String("tor-uri"))

	// Create the HTTP client
	httpClient := &fasthttp.Client{
		// Use given TOR proxy to reach the hidden services
		Dial: fasthttpproxy.FasthttpSocksDialer(ctx.String("tor-uri")),
		// Disable SSL verification since we do not really care about this
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	logrus.Info("Successfully initialized trandoshan-crawler. Waiting for URLs")

	if err := sub.QueueSubscribe(proto.URLTodoSubject, "crawlers", handleMessage(httpClient)); err != nil {
		return err
	}

	return nil
}

func handleMessage(httpClient *fasthttp.Client) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var urlMsg proto.URLTodoMsg
		if err := natsutil.ReadJSON(msg, &urlMsg); err != nil {
			return err
		}

		logrus.Debugf("Processing URL: %s", urlMsg.URL)

		// Query the website
		_, bytes, err := httpClient.Get(nil, urlMsg.URL)
		if err != nil {
			return err
		}
		body := string(bytes)

		// Publish resource body
		if err := natsutil.PublishJSON(nc, proto.ResourceSubject, &proto.ResourceMsg{URL: urlMsg.URL, Body: body}); err != nil {
			logrus.Warnf("Error while publishing resource body: %s", err)
		}

		// Extract URLs
		xu := xurls.Strict()
		urls := xu.FindAllString(body, -1)

		// Publish found URLs
		for _, url := range urls {
			logrus.Debugf("Found URL: %s", url)

			if err := natsutil.PublishJSON(nc, proto.URLFoundSubject, &proto.URLFoundMsg{URL: url}); err != nil {
				logrus.Warnf("Error while publishing URL: %s", err)
			}
		}

		return nil
	}
}
