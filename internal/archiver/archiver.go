package archiver

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/archiver/storage"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// GetApp return the crawler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-archiver",
		Version: "0.7.0",
		Usage:   "Trandoshan archiver component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetHubURI(),
			&cli.StringFlag{
				Name:     "storage-dir",
				Usage:    "Path to the storage directory",
				Required: true,
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
		Str("storage-dir", ctx.String("storage-dir")).
		Msg("Starting tdsh-archiver")

	// Create the subscriber
	sub, err := event.NewSubscriber(ctx.String("hub-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	// Create local storage
	st, err := storage.NewLocalStorage(ctx.String("storage-dir"))
	if err != nil {
		return err
	}

	state := state{
		storage: st,
	}

	if err := sub.SubscribeAsync(event.NewResourceExchange, "archivingQueue", state.handleNewResourceEvent); err != nil {
		return err
	}

	log.Info().Msg("Successfully initialized tdsh-archiver. Waiting for resources")

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
	storage storage.Storage
}

func (state *state) handleNewResourceEvent(subscriber event.Subscriber, body io.Reader) error {
	var evt event.NewResourceEvent
	if err := subscriber.Read(body, &evt); err != nil {
		return err
	}

	log.Debug().Str("url", evt.URL).Msg("Processing new resource")

	res, err := formatResource(&evt)
	if err != nil {
		return fmt.Errorf("error while formatting resource: %s", err)
	}

	if err := state.storage.Store(evt.URL, evt.Time, res); err != nil {
		return fmt.Errorf("error while storing resource: %s", err)
	}

	return nil
}

func formatResource(evt *event.NewResourceEvent) ([]byte, error) {
	builder := strings.Builder{}

	// First headers
	for key, value := range evt.Headers {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Then separator for body
	builder.WriteString("\r\n")

	// Then body
	builder.WriteString(evt.Body)

	return []byte(builder.String()), nil
}
