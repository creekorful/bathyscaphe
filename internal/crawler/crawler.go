package crawler

import (
	"crypto/tls"
	"github.com/creekorful/trandoshan/internal/util/log"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"mvdan.cc/xurls/v2"
	"strings"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0"

// GetApp return the crawler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-crawler",
		Version: "0.0.1",
		Usage:   "Trandoshan crawler process",
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
			&cli.StringFlag{
				Name:  "user-agent",
				Usage: "User agent to use",
				Value: defaultUserAgent,
			},
			&cli.StringSliceFlag{
				Name:  "allowed-ct",
				Usage: "Content types allowed to crawl",
				Value: cli.NewStringSlice("text/"),
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
	logrus.Debugf("Allowed content types: %s", ctx.StringSlice("allowed-ct"))

	// Create the HTTP client
	httpClient := &fasthttp.Client{
		// Use given TOR proxy to reach the hidden services
		Dial: fasthttpproxy.FasthttpSocksDialer(ctx.String("tor-uri")),
		// Disable SSL verification since we do not really care about this
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		Name:         ctx.String("user-agent"),
	}

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	logrus.Info("Successfully initialized trandoshan-crawler. Waiting for URLs")

	if err := sub.QueueSubscribe(proto.URLTodoSubject, "crawlers", handleMessage(httpClient, ctx.StringSlice("allowed-ct"))); err != nil {
		return err
	}

	return nil
}

func handleMessage(httpClient *fasthttp.Client, allowedContentTypes []string) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var urlMsg proto.URLTodoMsg
		if err := natsutil.ReadJSON(msg, &urlMsg); err != nil {
			return err
		}

		logrus.Debugf("Processing URL: %s", urlMsg.URL)

		// Query the website
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)
		defer fasthttp.ReleaseResponse(resp)

		req.SetRequestURI(urlMsg.URL)

		if err := httpClient.Do(req, resp); err != nil {
			logrus.Errorf("Error while querying website: %s", err)
			return err
		}

		// Determinate if content type is allowed
		allowed := false
		contentType := string(resp.Header.Peek("Content-Type"))
		for _, allowedContentType := range allowedContentTypes {
			if strings.Contains(contentType, allowedContentType) {
				allowed = true
				break
			}
		}

		if !allowed {
			logrus.Debugf("Discarding forbidden content type: %s", contentType)
			return nil
		}

		body := string(resp.Body())

		// Publish resource body
		res := proto.ResourceMsg{
			URL:  urlMsg.URL,
			Body: body,
		}
		if err := natsutil.PublishJSON(nc, proto.ResourceSubject, &res); err != nil {
			logrus.Errorf("Error while publishing resource body: %s", err)
		}

		// Extract URLs
		xu := xurls.Strict()
		urls := xu.FindAllString(body, -1)

		// Publish found URLs
		for _, url := range urls {
			logrus.Tracef("Found URL: %s", url)

			if err := natsutil.PublishJSON(nc, proto.URLFoundSubject, &proto.URLFoundMsg{URL: url}); err != nil {
				logrus.Errorf("Error while publishing URL: %s", err)
			}
		}

		return nil
	}
}
