package scheduler

import (
	"errors"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	errNotOnionHostname    = errors.New("hostname is not .onion")
	errProtocolNotAllowed  = errors.New("protocol is not allowed")
	errExtensionNotAllowed = errors.New("extension is not allowed")
	errShouldNotSchedule   = errors.New("should not be scheduled")
	errHostnameNotAllowed  = errors.New("hostname is not allowed")
)

// GetApp return the scheduler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-scheduler",
		Version: "0.7.0",
		Usage:   "Trandoshan scheduler component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetHubURI(),
			util.GetAPIURIFlag(),
			util.GetAPITokenFlag(),
			util.GetConfigAPIURIFlag(),
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
		Str("config-api-uri", ctx.String("config-api-uri")).
		Msg("Starting tdsh-scheduler")

	// Create the API client
	apiClient := util.GetAPIClient(ctx)

	// Create the subscriber
	sub, err := event.NewSubscriber(ctx.String("hub-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	// Create the ConfigAPI client
	keys := []string{client.ForbiddenMimeTypesKey, client.ForbiddenHostnamesKey, client.RefreshDelayKey}
	configClient, err := client.NewConfigClient(ctx.String("config-api-uri"), "scheduler", sub, keys)
	if err != nil {
		log.Err(err).Msg("error while creating config client")
		return err
	}

	state := state{
		apiClient:    apiClient,
		configClient: configClient,
	}

	if err := sub.SubscribeAsync(event.FoundURLExchange, "schedulingQueue", state.handleURLFoundEvent); err != nil {
		return err
	}

	log.Info().Msg("Successfully initialized tdsh-scheduler. Waiting for URLs")

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
	apiClient    api.API
	configClient client.Client
}

func (state *state) handleURLFoundEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.FoundURLEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	log.Trace().Str("url", evt.URL).Msg("Processing URL")

	u, err := url.Parse(evt.URL)
	if err != nil {
		return fmt.Errorf("error while parsing URL: %s", err)
	}

	// Make sure URL is valid .onion
	if !strings.Contains(u.Host, ".onion") {
		return fmt.Errorf("%s %w", u.Host, errNotOnionHostname)
	}

	// Make sure protocol is not forbidden
	if !strings.HasPrefix(u.Scheme, "http") {
		return fmt.Errorf("%s %w", u, errProtocolNotAllowed)
	}

	// Make sure extension is not forbidden
	if mimeTypes, err := state.configClient.GetForbiddenMimeTypes(); err == nil {
		for _, mimeType := range mimeTypes {
			for _, ext := range mimeType.Extensions {
				if strings.HasSuffix(u.Path, "."+ext) {
					return fmt.Errorf("%s (.%s) %w", u, ext, errExtensionNotAllowed)
				}
			}
		}
	}

	// Make sure hostname is not forbidden
	if hostnames, err := state.configClient.GetForbiddenHostnames(); err == nil {
		for _, hostname := range hostnames {
			if strings.Contains(u.Hostname(), hostname.Hostname) {
				return fmt.Errorf("%s %w", u, errHostnameNotAllowed)
			}
		}
	}

	// If we want to allow re-schedule of existing crawled resources we need to retrieve only resources
	// that are newer than `now - refreshDelay`.
	endDate := time.Time{}
	if refreshDelay, err := state.configClient.GetRefreshDelay(); err == nil {
		if refreshDelay.Delay != -1 {
			endDate = time.Now().Add(-refreshDelay.Delay)
		}
	}

	params := api.ResSearchParams{
		URL:        u.String(),
		EndDate:    endDate,
		WithBody:   false,
		PageSize:   1,
		PageNumber: 1,
	}
	_, count, err := state.apiClient.SearchResources(&params)
	if err != nil {
		return fmt.Errorf("error while searching resource (%s): %s", u, err)
	}

	if count > 0 {
		return fmt.Errorf("%s %w", u, errShouldNotSchedule)
	}

	// No matches: schedule!
	log.Debug().Stringer("url", u).Msg("URL should be scheduled")

	if err := subscriber.PublishEvent(&event.NewURLEvent{URL: evt.URL}); err != nil {
		return fmt.Errorf("error while publishing URL: %s", err)
	}

	return nil
}
