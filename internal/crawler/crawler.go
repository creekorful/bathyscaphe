package crawler

import (
	"crypto/tls"
	"fmt"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/util/logging"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
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
		Version: "0.3.0",
		Usage:   "Trandoshan crawler process",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
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
	logging.ConfigureLogger(ctx)

	log.Info().Str("ver", ctx.App.Version).Msg("Starting tdsh-crawler")

	log.Debug().Str("uri", ctx.String("nats-uri")).Msg("Using NATS server")
	log.Debug().Str("uri", ctx.String("tor-uri")).Msg("Using TOR proxy")
	log.Debug().Strs("content-types", ctx.StringSlice("allowed-ct")).Msg("Allowed content types")

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

	log.Info().Msg("Successfully initialized tdsh-crawler. Waiting for URLs")

	if err := sub.QueueSubscribe(messaging.URLTodoSubject, "crawlers",
		handleMessage(httpClient, ctx.StringSlice("allowed-ct"))); err != nil {
		return err
	}

	return nil
}

func handleMessage(httpClient *fasthttp.Client, allowedContentTypes []string) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var urlMsg messaging.URLTodoMsg
		if err := natsutil.ReadMsg(msg, &urlMsg); err != nil {
			return err
		}

		body, err := crawURL(httpClient, urlMsg.URL, allowedContentTypes)
		if err != nil {
			log.Err(err).Str("url", urlMsg.URL).Msg("Error while crawling url")
			return err
		}

		// Publish resource body
		res := messaging.NewResourceMsg{
			URL:  urlMsg.URL,
			Body: body,
		}
		if err := natsutil.PublishMsg(nc, &res); err != nil {
			log.Err(err).Msg("Error while publishing resource body")
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
