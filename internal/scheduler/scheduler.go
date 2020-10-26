package scheduler

import (
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/xhit/go-str2duration/v2"
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
			util.GetNATSURIFlag(),
			util.GetAPIURIFlag(),
			util.GetAPILoginFlag(),
			&cli.StringFlag{
				Name:  "refresh-delay",
				Usage: "Duration before allowing crawl of existing resource (none = never)",
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

	refreshDelay := parseRefreshDelay(ctx.String("refresh-delay"))
	if refreshDelay != -1 {
		log.Debug().Stringer("delay", refreshDelay).Msg("Existing resources will be crawled again")
	} else {
		log.Debug().Msg("Existing resources will NOT be crawled again")
	}

	// Create the API client
	apiClient, err := util.GetAPIAuthenticatedClient(ctx)
	if err != nil {
		return err
	}

	// Create the NATS subscriber
	sub, err := messaging.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	log.Info().Msg("Successfully initialized tdsh-scheduler. Waiting for URLs")

	if err := sub.QueueSubscribe(messaging.URLFoundSubject, "schedulers", handleMessage(apiClient, refreshDelay)); err != nil {
		return err
	}

	return nil
}

func handleMessage(apiClient api.Client, refreshDelay time.Duration) messaging.MsgHandler {
	return func(sub messaging.Subscriber, msg *nats.Msg) error {
		var urlMsg messaging.URLFoundMsg
		if err := sub.ReadMsg(msg, &urlMsg); err != nil {
			return err
		}

		log.Debug().Str("url", urlMsg.URL).Msg("Processing URL")

		u, err := url.Parse(urlMsg.URL)
		if err != nil {
			log.Err(err).Msg("Error while parsing URL")
			return err
		}

		// Make sure URL is valid .onion
		if !strings.Contains(u.Host, ".onion") {
			log.Debug().Stringer("url", u).Msg("URL is not a valid hidden service")
			return fmt.Errorf("%s is not a valid .onion", u.Host)
		}

		// If we want to allow re-schedule of existing crawled resources we need to retrieve only resources
		// that are newer than `now - refreshDelay`.
		endDate := time.Time{}
		if refreshDelay != -1 {
			endDate = time.Now().Add(-refreshDelay)
		}

		_, count, err := apiClient.SearchResources(u.String(), "", time.Time{}, endDate, 1, 1)
		if err != nil {
			log.Err(err).Msg("Error while searching URL")
			return err
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
