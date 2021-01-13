package indexer

import (
	"fmt"
	configapi "github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/constraint"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/indexer/index"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
)

var errHostnameNotAllowed = fmt.Errorf("hostname is not allowed")

// State represent the application state
type State struct {
	index        index.Index
	indexDriver  string
	configClient configapi.Client

	bufferThreshold int
	resources       []index.Resource
}

// Name return the process name
func (state *State) Name() string {
	return "indexer"
}

// Description return the process description
func (state *State) Description() string {
	return `
The indexing component. It consumes crawled resources, format
them and finally index them using the configured driver.

This component consumes the 'resource.new' event.`
}

// Features return the process features
func (state *State) Features() []process.Feature {
	return []process.Feature{process.EventFeature, process.ConfigFeature}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "index-driver",
			Usage:    "Name of the storage driver",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "index-dest",
			Usage:    "Destination (config) passed to the driver",
			Required: true,
		},
	}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	indexDriver := provider.GetStrValue("index-driver")
	idx, err := index.NewIndex(indexDriver, provider.GetStrValue("index-dest"))
	if err != nil {
		return err
	}
	state.index = idx
	state.indexDriver = indexDriver
	state.bufferThreshold = provider.GetIntValue(process.EventPrefetchFlag)

	configClient, err := provider.ConfigClient([]string{configapi.ForbiddenHostnamesKey})
	if err != nil {
		return err
	}
	state.configClient = configClient

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.NewResourceExchange, Queue: fmt.Sprintf("%sIndexingQueue", state.indexDriver), Handler: state.handleNewResourceEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler() http.Handler {
	return nil
}

func (state *State) handleNewResourceEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewResourceEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	// make sure hostname hasn't been flagged as forbidden
	if allowed, err := constraint.CheckHostnameAllowed(state.configClient, evt.URL); !allowed || err != nil {
		return fmt.Errorf("%s %w", evt.URL, errHostnameNotAllowed)
	}

	// Direct saving (no buffering)
	if state.bufferThreshold == 1 {
		if err := state.index.IndexResource(index.Resource{
			URL:     evt.URL,
			Time:    evt.Time,
			Body:    evt.Body,
			Headers: evt.Headers,
		}); err != nil {
			return fmt.Errorf("error while indexing resource: %s", err)
		}

		log.Info().
			Str("url", evt.URL).
			Msg("Successfully indexed resource")

		return nil
	}

	// Otherwise we are in buffered saving mode
	state.resources = append(state.resources, index.Resource{
		URL:     evt.URL,
		Time:    evt.Time,
		Body:    evt.Body,
		Headers: evt.Headers,
	})

	log.Debug().Str("url", evt.URL).Msg("Successfully stored resource in buffer")

	if len(state.resources) >= state.bufferThreshold {
		// Time to save!
		if err := state.index.IndexResources(state.resources); err != nil {
			return fmt.Errorf("error while indexing resources: %s", err)
		}

		log.Info().
			Int("count", len(state.resources)).
			Msg("Successfully indexed buffered resources")

		// Clear cache
		state.resources = []index.Resource{}
	}

	return nil
}
