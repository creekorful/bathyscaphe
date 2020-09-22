package extractor

import (
	"fmt"
	"github.com/PuerkitoBio/purell"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/util/logging"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"mvdan.cc/xurls/v2"
	"regexp"
	"strings"
	"time"
)

var (
	protocolRegex = regexp.MustCompile("https?://")
)

// GetApp return the extractor app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-extractor",
		Version: "0.3.0",
		Usage:   "Trandoshan extractor process",
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

	log.Info().Str("ver", ctx.App.Version).Msg("Starting tdsh-extractor")

	log.Debug().Str("uri", ctx.String("nats-uri")).Msg("Using NATS server")
	log.Debug().Str("uri", ctx.String("api-uri")).Msg("Using API server")

	// Create the API client
	apiClient := api.NewClient(ctx.String("api-uri"))

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	log.Info().Msg("Successfully initialized tdsh-extractor. Waiting for resources")

	if err := sub.QueueSubscribe(messaging.NewResourceSubject, "extractors",
		handleMessage(apiClient)); err != nil {
		return err
	}

	return nil
}

func handleMessage(apiClient api.Client) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var resMsg messaging.NewResourceMsg
		if err := natsutil.ReadMsg(msg, &resMsg); err != nil {
			log.Err(err).Msg("Error while reading message")
			return err
		}

		log.Debug().Str("url", resMsg.URL).Msg("Processing new resource")

		// Extract & process resource
		resDto, urls, err := extractResource(resMsg)
		if err != nil {
			log.Err(err).Msg("Ersror while extracting resource")
			return err
		}

		// Submit to the API
		_, err = apiClient.AddResource(resDto)
		if err != nil {
			log.Err(err).Msg("Error while adding resource")
			return err
		}

		// Finally push found URLs
		for _, url := range urls {
			log.Trace().
				Str("url", url).
				Msg("Publishing found URL")

			if err := natsutil.PublishMsg(nc, &messaging.URLFoundMsg{URL: url}); err != nil {
				log.Warn().
					Str("url", url).
					Str("err", err.Error()).
					Msg("Error while publishing URL")
			}
		}

		return nil
	}
}

func extractResource(msg messaging.NewResourceMsg) (api.ResourceDto, []string, error) {
	resDto := api.ResourceDto{
		URL:   protocolRegex.ReplaceAllLiteralString(msg.URL, ""),
		Title: extractTitle(msg.Body),
		Body:  msg.Body,
		Time:  time.Now(),
	}

	// Extract URLs
	xu := xurls.Strict()

	// Sanitize URLs
	urls := xu.FindAllString(msg.Body, -1)
	var normalizedURLS []string

	for _, url := range urls {
		normalizedURL, err := normalizeURL(url)
		if err != nil {
			continue
		}

		normalizedURLS = append(normalizedURLS, normalizedURL)
	}

	return resDto, normalizedURLS, nil
}

// extract title from html body
func extractTitle(body string) string {
	cleanBody := strings.ToLower(body)

	if strings.Index(cleanBody, "<title>") == -1 || strings.Index(cleanBody, "</title>") == -1 {
		return ""
	}

	// TODO improve
	startPos := strings.Index(cleanBody, "<title>") + len("<title>")
	endPos := strings.Index(cleanBody, "</title>")

	return body[startPos:endPos]
}

func normalizeURL(u string) (string, error) {
	normalizedURL, err := purell.NormalizeURLString(u, purell.FlagsUsuallySafeGreedy|
		purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)
	if err != nil {
		return "", fmt.Errorf("error while normalizing URL %s: %s", u, err)
	}

	return normalizedURL, nil
}
