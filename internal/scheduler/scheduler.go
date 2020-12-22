package scheduler

import (
	"errors"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/duration"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	ErrNotOnionHostname    = errors.New("hostname is not .onion")
	ErrProtocolNotAllowed  = errors.New("protocol is not allowed")
	ErrExtensionNotAllowed = errors.New("extension is not allowed")
	ErrShouldNotSchedule   = errors.New("should not be scheduled")
)

// GetApp return the scheduler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-scheduler",
		Version: "0.6.0",
		Usage:   "Trandoshan scheduler component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetHubURI(),
			util.GetAPIURIFlag(),
			util.GetAPITokenFlag(),
			&cli.StringFlag{
				Name:  "refresh-delay",
				Usage: "Duration before allowing crawl of existing resource (none = never)",
			},
			&cli.StringSliceFlag{
				Name:  "forbidden-extensions",
				Usage: "Extensions to disable scheduling for (i.e png, exe, css, ...) (the dot will be added automatically)",
			},
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)

	refreshDelay := duration.ParseDuration(ctx.String("refresh-delay"))

	log.Info().
		Str("ver", ctx.App.Version).
		Str("hub-uri", ctx.String("hub-uri")).
		Str("api-uri", ctx.String("api-uri")).
		Strs("forbidden-exts", ctx.StringSlice("forbidden-extensions")).
		Dur("refresh-delay", refreshDelay).
		Msg("Starting tdsh-scheduler")

	// Create the API client
	apiClient := util.GetAPIClient(ctx)

	// Create the subscriber
	sub, err := event.NewSubscriber(ctx.String("hub-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	state := state{
		apiClient:           apiClient,
		refreshDelay:        refreshDelay,
		forbiddenExtensions: ctx.StringSlice("forbidden-extensions"),
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
	apiClient           api.Client
	refreshDelay        time.Duration
	forbiddenExtensions []string
}

func (state *state) handleURLFoundEvent(subscriber event.Subscriber, body io.Reader) error {
	var evt event.FoundURLEvent
	if err := subscriber.Read(body, &evt); err != nil {
		return err
	}

	log.Trace().Str("url", evt.URL).Msg("Processing URL")

	u, err := url.Parse(evt.URL)
	if err != nil {
		return fmt.Errorf("error while parsing URL: %s", err)
	}

	// Make sure URL is valid .onion
	if !strings.Contains(u.Host, ".onion") {
		return fmt.Errorf("%s %w", u.Host, ErrNotOnionHostname)
	}

	// Make sure protocol is allowed
	if !strings.HasPrefix(u.Scheme, "http") {
		return fmt.Errorf("%s %w", u, ErrProtocolNotAllowed)
	}

	// Make sure extension is not forbidden
	for _, ext := range state.forbiddenExtensions {
		if strings.HasSuffix(u.Path, "."+ext) {
			return fmt.Errorf("%s (.%s) %w", u, ext, ErrExtensionNotAllowed)
		}
	}

	// If we want to allow re-schedule of existing crawled resources we need to retrieve only resources
	// that are newer than `now - refreshDelay`.
	endDate := time.Time{}
	if state.refreshDelay != -1 {
		endDate = time.Now().Add(-state.refreshDelay)
	}

	_, count, err := state.apiClient.SearchResources(u.String(), "", time.Time{}, endDate, 1, 1)
	if err != nil {
		return fmt.Errorf("error while searching resource (%s): %s", u, err)
	}

	if count > 0 {
		return fmt.Errorf("%s %w", u, ErrShouldNotSchedule)
	}

	// No matches: schedule!
	log.Debug().Stringer("url", u).Msg("URL should be scheduled")

	if err := subscriber.Publish(&event.NewURLEvent{URL: evt.URL}); err != nil {
		return fmt.Errorf("error while publishing URL: %s", err)
	}

	return nil
}
