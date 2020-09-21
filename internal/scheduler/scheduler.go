package scheduler

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/purell"
	"github.com/creekorful/trandoshan/internal/util/http"
	"github.com/creekorful/trandoshan/internal/util/logging"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/url"
	"strings"
)

// GetApp return the scheduler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-scheduler",
		Version: "0.3.0",
		Usage:   "Trandoshan scheduler process",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			&cli.StringFlag{
				Name:     "nats-uri",
				Usage:    "URI to the NATS server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "api-uri",
				Usage:    "URI to the API server",
				Required: true,
			},
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)

	log.Info().Str("ver", ctx.App.Version).Msg("Starting tdsh-scheduler")

	log.Debug().Str("uri", ctx.String("nats-uri")).Msg("Using NATS server")
	log.Debug().Str("uri", ctx.String("api-uri")).Msg("Using API server")

	// Create the HTTP client
	httpClient := &http.Client{}

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	log.Info().Msg("Successfully initialized tdsh-scheduler. Waiting for URLs")

	if err := sub.QueueSubscribe(proto.URLFoundSubject, "schedulers", handleMessage(httpClient, ctx.String("api-uri"))); err != nil {
		return err
	}

	return nil
}

func handleMessage(httpClient *http.Client, apiURI string) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var urlMsg proto.URLFoundMsg
		if err := natsutil.ReadJSON(msg, &urlMsg); err != nil {
			return err
		}

		log.Debug().Str("url", urlMsg.URL).Msg("Processing URL: %s")
		normalizedURL, err := normalizeURL(urlMsg.URL)
		if err != nil {
			log.Err(err).Msg("Error while normalizing URL")
			return err
		}

		// Make sure URL is valid .onion
		if !strings.Contains(normalizedURL.Host, ".onion") {
			log.Debug().Stringer("url", normalizedURL).Msg("URL is not a valid hidden service")
			return err
		}

		b64URI := base64.URLEncoding.EncodeToString([]byte(normalizedURL.String()))
		apiURL := fmt.Sprintf("%s/v1/resources?url=%s", apiURI, b64URI)

		var urls []proto.ResourceDto
		_, err = httpClient.JSONGet(apiURL, &urls)
		if err != nil {
			log.Err(err).Msg("Error while searching URL")
			return err
		}

		// No matches: schedule!
		if len(urls) == 0 {
			log.Debug().Stringer("url", normalizedURL).Msg("URL should be scheduled")
			if err := natsutil.PublishJSON(nc, proto.URLTodoSubject, &proto.URLTodoMsg{URL: urlMsg.URL}); err != nil {
				return fmt.Errorf("error while publishing URL: %s", err)
			}
		} else {
			log.Trace().Stringer("url", normalizedURL).Msg("URL should not be scheduled")
		}

		return nil
	}
}

func normalizeURL(u string) (*url.URL, error) {
	normalizedURL, err := purell.NormalizeURLString(u, purell.FlagsUsuallySafeGreedy|
		purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)
	if err != nil {
		return nil, fmt.Errorf("error while normalizing URL %s: %s", u, err)
	}

	nu, err := url.Parse(normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("error while parsing URL: %s", err)
	}

	return nu, nil
}
