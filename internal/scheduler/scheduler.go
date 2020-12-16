package scheduler

import (
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/xhit/go-str2duration/v2"
	"io"
	"net/url"
	"strings"
	"time"
)

// GetApp return the scheduler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-scheduler",
		Version: "0.5.1",
		Usage:   "Trandoshan scheduler component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetEventSrvURI(),
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

	refreshDelay := parseRefreshDelay(ctx.String("refresh-delay"))

	log.Info().
		Str("ver", ctx.App.Version).
		Str("event-srv-uri", ctx.String("event-srv-uri")).
		Str("api-uri", ctx.String("api-uri")).
		Strs("forbidden-exts", ctx.StringSlice("forbidden-extensions")).
		Dur("refresh-delay", refreshDelay).
		Msg("Starting tdsh-scheduler")

	// Create the API client
	apiClient := util.GetAPIClient(ctx)

	// Create the subscriber
	sub, err := messaging.NewSubscriber(ctx.String("event-srv-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	log.Info().Msg("Successfully initialized tdsh-scheduler. Waiting for URLs")

	handler := handleMessage(apiClient, refreshDelay, ctx.StringSlice("forbidden-extensions"))
	if err := sub.QueueSubscribe(messaging.URLFoundSubject, "schedulers", handler); err != nil {
		return err
	}

	return nil
}

func handleMessage(apiClient api.Client, refreshDelay time.Duration, forbiddenExtensions []string) messaging.MsgHandler {
	return func(sub messaging.Subscriber, msg io.Reader) error {
		var urlMsg messaging.URLFoundMsg
		if err := sub.ReadMsg(msg, &urlMsg); err != nil {
			return err
		}

		log.Trace().Str("url", urlMsg.URL).Msg("Processing URL")

		u, err := url.Parse(urlMsg.URL)
		if err != nil {
			return fmt.Errorf("error while parsing URL: %s", err)
		}

		// Make sure URL is valid .onion
		if !strings.Contains(u.Host, ".onion") {
			log.Trace().Stringer("url", u).Msg("URL is not a valid hidden service")
			return nil // Technically not an error
		}

		// Make sure extension is not forbidden
		for _, ext := range forbiddenExtensions {
			if strings.HasSuffix(u.Path, "."+ext) {
				log.Trace().
					Stringer("url", u).
					Str("ext", ext).
					Msg("Skipping URL with forbidden extension")
				return nil // Technically not an error
			}
		}

		// If we want to allow re-schedule of existing crawled resources we need to retrieve only resources
		// that are newer than `now - refreshDelay`.
		endDate := time.Time{}
		if refreshDelay != -1 {
			endDate = time.Now().Add(-refreshDelay)
		}

		_, count, err := apiClient.SearchResources(u.String(), "", time.Time{}, endDate, 1, 1)
		if err != nil {
			return fmt.Errorf("error while searching resource (%s): %s", u, err)
		}

		// No matches: schedule!
		if count == 0 {
			log.Debug().Stringer("url", u).Msg("URL should be scheduled")
			if err := sub.PublishMsg(&messaging.URLTodoMsg{URL: urlMsg.URL}); err != nil {
				return fmt.Errorf("error while publishing URL: %s", err)
			}
		} else {
			log.Trace().Stringer("url", u).Msg("URL should not be scheduled")
		}

		return nil
	}
}

func parseRefreshDelay(delay string) time.Duration {
	if delay == "" {
		return -1
	}

	val, err := str2duration.ParseDuration(delay)
	if err != nil {
		return -1
	}

	return val
}
