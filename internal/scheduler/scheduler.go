package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/constraint"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
	"strings"
)

var (
	errNotOnionHostname    = errors.New("hostname is not .onion")
	errProtocolNotAllowed  = errors.New("protocol is not allowed")
	errExtensionNotAllowed = errors.New("extension is not allowed")
	errHostnameNotAllowed  = errors.New("hostname is not allowed")
)

// State represent the application state
type State struct {
	configClient configapi.Client
	pub          event.Publisher
}

// Name return the process name
func (state *State) Name() string {
	return "scheduler"
}

// CommonFlags return process common flags
func (state *State) CommonFlags() []string {
	return []string{process.HubURIFlag, process.ConfigAPIURIFlag}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	keys := []string{configapi.ForbiddenMimeTypesKey, configapi.ForbiddenHostnamesKey}
	configClient, err := provider.ConfigClient(keys)
	if err != nil {
		return err
	}
	state.configClient = configClient

	pub, err := provider.Publisher()
	if err != nil {
		return err
	}
	state.pub = pub

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.FoundURLExchange, Queue: "schedulingQueue", Handler: state.handleURLFoundEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler(_ process.Provider) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/urls", state.scheduleURL).Methods(http.MethodPost)

	return r
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

	log.Debug().Stringer("url", u).Msg("URL should be scheduled")

	if err := subscriber.PublishEvent(&event.NewURLEvent{URL: evt.URL}); err != nil {
		return fmt.Errorf("error while publishing URL: %s", err)
	}

	return nil
}

func (state *State) scheduleURL(w http.ResponseWriter, r *http.Request) {
	var u string
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		log.Warn().Str("err", err.Error()).Msg("error while decoding request body")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if err := state.pub.PublishEvent(&event.FoundURLEvent{URL: u}); err != nil {
		log.Err(err).Msg("unable to schedule URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Info().Str("url", u).Msg("successfully scheduled URL")
}
