package archiver

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/archiver/storage"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
	"strings"
)

type State struct {
	storage storage.Storage
}

func (state *State) Name() string {
	return "archiver"
}

func (state *State) CommonFlags() []string {
	return []string{process.HubURIFlag}
}

func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "storage-dir",
			Usage:    "Path to the storage directory",
			Required: true,
		},
	}
}

func (state *State) Provide(provider process.Provider) error {
	st, err := storage.NewLocalStorage(provider.GetValue("storage-dir"))
	if err != nil {
		return err
	}
	state.storage = st

	return nil
}

func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.NewResourceExchange, Queue: "archivingQueue", Handler: state.handleNewResourceEvent},
	}
}

func (state *State) HTTPHandler() http.Handler {
	return nil
}

func (state *State) handleNewResourceEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewResourceEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
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
