package crawler

import (
	"crypto/tls"
	"fmt"
	"github.com/creekorful/trandoshan/internal/http"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0"

// GetApp return the crawler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-crawler",
		Version: "0.5.1",
		Usage:   "Trandoshan crawler component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetEventSrvURI(),
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
		Str("event-srv-uri", ctx.String("event-srv-uri")).
		Str("tor-uri", ctx.String("tor-uri")).
		Strs("allowed-content-types", ctx.StringSlice("allowed-ct")).
		Msg("Starting tdsh-crawler")

	// Create the HTTP client
	httpClient := http.NewFastHTTPClient(&fasthttp.Client{
		// Use given TOR proxy to reach the hidden services
		Dial: fasthttpproxy.FasthttpSocksDialer(ctx.String("tor-uri")),
		// Disable SSL verification since we do not really care about this
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		Name:         ctx.String("user-agent"),
	})

	// Create the subscriber
	sub, err := messaging.NewSubscriber(ctx.String("event-srv-uri"))
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

func handleMessage(httpClient http.Client, allowedContentTypes []string) messaging.MsgHandler {
	return func(sub messaging.Subscriber, msg io.Reader) error {
		var urlMsg messaging.URLTodoMsg
		if err := sub.ReadMsg(msg, &urlMsg); err != nil {
			return err
		}

		body, headers, err := crawURL(httpClient, urlMsg.URL, allowedContentTypes)
		if err != nil {
			return fmt.Errorf("error while crawling URL: %s", err)
		}

		// Publish resource body
		res := messaging.NewResourceMsg{
			URL:     urlMsg.URL,
			Body:    body,
			Headers: headers,
		}
		if err := sub.PublishMsg(&res); err != nil {
			return fmt.Errorf("error while publishing resource: %s", err)
		}

		return nil
	}
}

func crawURL(httpClient http.Client, url string, allowedContentTypes []string) (string, map[string]string, error) {
	log.Debug().Str("url", url).Msg("Processing URL")

	r, err := httpClient.Get(url)
	if err != nil {
		return "", nil, err
	}

	// Determinate if content type is allowed
	allowed := false
	contentType := r.Headers()["Content-Type"]
	for _, allowedContentType := range allowedContentTypes {
		if strings.Contains(contentType, allowedContentType) {
			allowed = true
			break
		}
	}

	if !allowed {
		err := fmt.Errorf("forbidden content type : %s", contentType)
		return "", nil, err
	}

	// Ready body
	b, err := ioutil.ReadAll(r.Body())
	if err != nil {
		return "", nil, err
	}
	return string(b), r.Headers(), nil
}
