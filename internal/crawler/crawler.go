package crawler

import (
	"crypto/tls"
	"fmt"
	"github.com/creekorful/trandoshan/internal/clock"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/crawler/http"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0"

// GetApp return the crawler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-crawler",
		Version: "0.7.0",
		Usage:   "Trandoshan crawler component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetHubURI(),
			util.GetConfigAPIURIFlag(),
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
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)

	log.Info().
		Str("ver", ctx.App.Version).
		Str("hub-uri", ctx.String("hub-uri")).
		Str("tor-uri", ctx.String("tor-uri")).
		Str("config-api-uri", ctx.String("config-api-uri")).
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
	sub, err := event.NewSubscriber(ctx.String("hub-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	// Create the ConfigAPI client
	keys := []string{client.AllowedMimeTypesKey}
	configClient, err := client.NewConfigClient(ctx.String("config-api-uri"), sub, keys)
	if err != nil {
		log.Err(err).Msg("error while creating config client")
		return err
	}

	state := state{
		httpClient:   httpClient,
		clock:        &clock.SystemClock{},
		configClient: configClient,
	}

	if err := sub.Subscribe(event.NewURLExchange, "crawlingQueue", state.handleNewURLEvent); err != nil {
		return err
	}

	log.Info().Msg("Successfully initialized tdsh-crawler. Waiting for URLs")

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-c

	return nil
}

type state struct {
	httpClient   http.Client
	clock        clock.Clock
	configClient client.Client
}

func (state *state) handleNewURLEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewURLEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	b, headers, err := crawURL(state.httpClient, evt.URL, state.configClient)
	if err != nil {
		return err
	}

	res := event.NewResourceEvent{
		URL:     evt.URL,
		Body:    b,
		Headers: headers,
		Time:    state.clock.Now(),
	}

	if err := subscriber.PublishEvent(&res); err != nil {
		return err
	}

	return nil
}

func crawURL(httpClient http.Client, url string, configClient client.Client) (string, map[string]string, error) {
	log.Debug().Str("url", url).Msg("Processing URL")

	r, err := httpClient.Get(url)
	if err != nil {
		return "", nil, err
	}

	// Determinate if content type is allowed
	allowed := false
	contentType := r.Headers()["Content-Type"]

	if allowedMimeTypes, err := configClient.GetAllowedMimeTypes(); err == nil {
		if len(allowedMimeTypes) == 0 {
			allowed = true
		}

		for _, allowedMimeType := range allowedMimeTypes {
			if strings.Contains(contentType, allowedMimeType.ContentType) {
				allowed = true
				break
			}
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
