package scheduler

import (
	"errors"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/constraint"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	errNotOnionHostname    = errors.New("hostname is not .onion")
	errProtocolNotAllowed  = errors.New("protocol is not allowed")
	errExtensionNotAllowed = errors.New("extension is not allowed")
	errShouldNotSchedule   = errors.New("should not be scheduled")
	errHostnameNotAllowed  = errors.New("hostname is not allowed")
)

// State represent the application state
type State struct {
	apiClient    api.API
	configClient configapi.Client
}

// Name return the process name
func (state *State) Name() string {
	return "scheduler"
}

// CommonFlags return process common flags
func (state *State) CommonFlags() []string {
	return []string{process.HubURIFlag, process.APIURIFlag, process.APITokenFlag, process.ConfigAPIURIFlag}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	apiClient, err := provider.APIClient()
	if err != nil {
		return err
	}
	state.apiClient = apiClient

	keys := []string{configapi.ForbiddenMimeTypesKey, configapi.ForbiddenHostnamesKey, configapi.RefreshDelayKey}
	configClient, err := provider.ConfigClient(keys)
	if err != nil {
		return err
	}
	state.configClient = configClient

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.FoundURLExchange, Queue: "schedulingQueue", Handler: state.handleURLFoundEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler(provider process.Provider) http.Handler {
	return nil
}

func (state *State) handleURLFoundEvent(subscriber event.Subscriber, msg event.RawMessage) error {
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
	if !strings.HasSuffix(u.Hostname(), ".onion") {
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
				if strings.HasSuffix(strings.ToLower(u.Path), "."+ext) {
					return fmt.Errorf("%s (.%s) %w", u, ext, errExtensionNotAllowed)
				}
			}
		}
	}

	// Make sure hostname is not forbidden
	if allowed, err := constraint.CheckHostnameAllowed(state.configClient, evt.URL); err != nil {
		return err
	} else if !allowed {
		log.Debug().Str("url", evt.URL).Msg("Skipping forbidden hostname")
		return fmt.Errorf("%s %w", u, errHostnameNotAllowed)
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
