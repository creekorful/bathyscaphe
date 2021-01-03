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

// State represent the application state
type State struct {
	storage storage.Storage
}

// Name return the process name
func (state *State) Name() string {
	return "archiver"
}

// CommonFlags return process common flags
func (state *State) CommonFlags() []string {
	return []string{process.HubURIFlag}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "storage-dir",
			Usage:    "Path to the storage directory",
			Required: true,
		},
	}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	st, err := storage.NewLocalStorage(provider.GetValue("storage-dir"))
	if err != nil {
		return err
	}
	state.storage = st

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.NewIndexExchange, Queue: "archivingQueue", Handler: state.handleNewIndexEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler(provider process.Provider) http.Handler {
	return nil
}

func (state *State) handleNewIndexEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewIndexEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	res, err := formatResource(&evt)
	if err != nil {
		return fmt.Errorf("error while formatting resource: %s", err)
	}

	if err := state.storage.Store(evt.URL, evt.Time, res); err != nil {
		return fmt.Errorf("error while storing resource: %s", err)
	}

	log.Debug().Str("url", evt.URL).Msg("Successfully archived resource")

	return nil
}

func formatResource(evt *event.NewIndexEvent) ([]byte, error) {
	builder := strings.Builder{}

	// First URL
	builder.WriteString(fmt.Sprintf("%s\n\n", evt.URL))

	// Then headers
	for key, value := range evt.Headers {
		builder.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}
	builder.WriteString("\n")

	// Then body
	builder.WriteString(evt.Body)

	return []byte(builder.String()), nil
}
