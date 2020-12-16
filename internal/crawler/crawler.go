package crawler

import (
	"crypto/tls"
	"fmt"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"strings"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0"

// GetApp return the crawler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-crawler",
		Version: "0.6.0",
		Usage:   "Trandoshan crawler component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetNATSURIFlag(),
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
	logging.ConfigureLogger(ctx)

	log.Info().
		Str("ver", ctx.App.Version).
		Str("nats-uri", ctx.String("nats-uri")).
		Str("tor-uri", ctx.String("tor-uri")).
		Strs("allowed-content-types", ctx.StringSlice("allowed-ct")).
		Msg("Starting tdsh-crawler")

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
	sub, err := messaging.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	log.Info().Msg("Successfully initialized tdsh-crawler. Waiting for URLs")

	handler := handleMessage(httpClient, ctx.StringSlice("allowed-ct"))
	if err := sub.QueueSubscribe(messaging.URLTodoSubject, "crawlers", handler); err != nil {
		return err
	}

	return nil
}

func handleMessage(httpClient *fasthttp.Client, allowedContentTypes []string) messaging.MsgHandler {
	return func(sub messaging.Subscriber, msg *nats.Msg) error {
		var urlMsg messaging.URLTodoMsg
		if err := sub.ReadMsg(msg, &urlMsg); err != nil {
			return err
		}

		body, err := crawURL(httpClient, urlMsg.URL, allowedContentTypes)
		if err != nil {
			return fmt.Errorf("error while crawling URL: %s", err)
		}

		// Publish resource body
		res := messaging.NewResourceMsg{
			URL:  urlMsg.URL,
			Body: body,
		}
		if err := sub.PublishMsg(&res); err != nil {
			return fmt.Errorf("error while publishing resource: %s", err)
		}

		return nil
	}
}

func crawURL(httpClient *fasthttp.Client, url string, allowedContentTypes []string) (string, error) {
	log.Debug().Str("url", url).Msg("Processing URL")

	// Query the website
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)

	if err := httpClient.Do(req, resp); err != nil {
		return "", err
	}

	switch code := resp.StatusCode(); {
	case code > 302:
		return "", fmt.Errorf("non-managed error code %d", code)
	// follow redirect
	case code == 301 || code == 302:
		if location := string(resp.Header.Peek("Location")); location != "" {
			return crawURL(httpClient, location, allowedContentTypes)
		}
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
		err := fmt.Errorf("forbidden content type : %s", contentType)
		return "", err
	}

	return string(resp.Body()), nil
}
