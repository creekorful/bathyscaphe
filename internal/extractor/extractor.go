package extractor

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io"
	"mvdan.cc/xurls/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// GetApp return the extractor app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-extractor",
		Version: "0.6.0",
		Usage:   "Trandoshan extractor component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetHubURI(),
			util.GetAPIURIFlag(),
			util.GetAPITokenFlag(),
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)

	log.Info().
		Str("ver", ctx.App.Version).
		Str("hub-uri", ctx.String("hub-uri")).
		Str("api-uri", ctx.String("api-uri")).
		Msg("Starting tdsh-extractor")

	apiClient := util.GetAPIClient(ctx)

	// Create the event subscriber
	sub, err := event.NewSubscriber(ctx.String("hub-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	state := state{apiClient: apiClient}

	if err := sub.SubscribeAsync(event.NewResourceExchange, "extractingQueue", state.handleNewResourceEvent); err != nil {
		return err
	}

	log.Info().Msg("Successfully initialized tdsh-extractor. Waiting for resources")

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-c

	if err := sub.Close(); err != nil {
		return err
	}

	return nil
}

type state struct {
	apiClient api.Client
}

func (state *state) handleNewResourceEvent(subscriber event.Subscriber, body io.Reader) error {
	var evt event.NewResourceEvent
	if err := subscriber.Read(body, &evt); err != nil {
		return err
	}

	log.Debug().Str("url", evt.URL).Msg("Processing new resource")

	// Extract & process resource
	resDto, urls, err := extractResource(evt)
	if err != nil {
		return fmt.Errorf("error while extracting resource: %s", err)
	}

	// Lowercase headers
	resDto.Headers = map[string]string{}
	for key, value := range evt.Headers {
		resDto.Headers[strings.ToLower(key)] = value
	}

	// Submit to the API
	_, err = state.apiClient.AddResource(resDto)
	if err != nil {
		return fmt.Errorf("error while adding resource (%s): %s", resDto.URL, err)
	}

	// Finally push found URLs
	publishedURLS := map[string]string{}
	for _, url := range urls {
		if _, exist := publishedURLS[url]; exist {
			log.Trace().
				Str("url", url).
				Msg("Skipping duplicate URL")
			continue
		}

		log.Trace().
			Str("url", url).
			Msg("Publishing found URL")

		if err := subscriber.Publish(&event.FoundURLEvent{URL: url}); err != nil {
			log.Warn().
				Str("url", url).
				Str("err", err.Error()).
				Msg("Error while publishing URL")
		}

		publishedURLS[url] = url
	}

	return nil
}

func extractResource(msg event.NewResourceEvent) (api.ResourceDto, []string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(msg.Body))
	if err != nil {
		return api.ResourceDto{}, nil, err
	}

	// Get resource title
	title := doc.Find("title").First().Text()

	// Get meta values
	meta := map[string]string{}
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		value, _ := s.Attr("content")

		// if name is empty then try to lookup using property
		if name == "" {
			name, _ = s.Attr("property")
			if name == "" {
				return
			}
		}

		meta[strings.ToLower(name)] = value
	})

	// Extract & normalize URLs
	xu := xurls.Strict()
	urls := xu.FindAllString(msg.Body, -1)

	var normalizedURLS []string

	for _, url := range urls {
		normalizedURL, err := normalizeURL(url)
		if err != nil {
			continue
		}

		normalizedURLS = append(normalizedURLS, normalizedURL)
	}

	return api.ResourceDto{
		URL:         msg.URL,
		Body:        msg.Body,
		Time:        msg.Time,
		Title:       title,
		Meta:        meta,
		Description: meta["description"],
	}, normalizedURLS, nil
}

func normalizeURL(u string) (string, error) {
	normalizedURL, err := purell.NormalizeURLString(u, purell.FlagsUsuallySafeGreedy|
		purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)
	if err != nil {
		return "", fmt.Errorf("error while normalizing URL %s: %s", u, err)
	}

	return normalizedURL, nil
}
