package crawler

import (
	"fmt"
	"github.com/darkspot-org/bathyscaphe/internal/clock"
	configapi "github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/constraint"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	chttp "github.com/darkspot-org/bathyscaphe/internal/http"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	errContentTypeNotAllowed = fmt.Errorf("content type is not allowed")
	errHostnameNotAllowed    = fmt.Errorf("hostname is not allowed")
)

// State represent the application state
type State struct {
	httpClient   chttp.Client
	clock        clock.Clock
	configClient configapi.Client
}

// Name return the process name
func (state *State) Name() string {
	return "crawler"
}

// Description return the process description
func (state *State) Description() string {
	return `
The crawling component. It consumes URL, crawl the resource, and
publish the result (page content + headers).

The crawler consumes the 'url.new' event and produces either:
- 'url.timeout' event if the crawling has failed because of timeout issue
- 'resource.new' event if the crawling has succeeded.`
}

// Features return the process features
func (state *State) Features() []process.Feature {
	return []process.Feature{process.EventFeature, process.ConfigFeature, process.CrawlingFeature}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	httpClient, err := provider.HTTPClient()
	if err != nil {
		return err
	}
	state.httpClient = httpClient

	cl, err := provider.Clock()
	if err != nil {
		return err
	}
	state.clock = cl

	configClient, err := provider.ConfigClient([]string{configapi.AllowedMimeTypesKey, configapi.ForbiddenHostnamesKey})
	if err != nil {
		return err
	}
	state.configClient = configClient

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.NewURLExchange, Queue: "crawlingQueue", Handler: state.handleNewURLEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler() http.Handler {
	return nil
}

func (state *State) handleNewURLEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewURLEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	log.Debug().Str("url", evt.URL).Msg("Processing URL")

	if allowed, err := constraint.CheckHostnameAllowed(state.configClient, evt.URL); err != nil {
		return err
	} else if !allowed {
		log.Debug().Str("url", evt.URL).Msg("Skipping forbidden hostname")
		return fmt.Errorf("%s %w", evt.URL, errHostnameNotAllowed)
	}

	r, err := state.httpClient.Get(evt.URL)
	if err != nil {
		if err == chttp.ErrTimeout {
			// indicate that crawling has failed
			_ = subscriber.PublishEvent(&event.TimeoutURLEvent{URL: evt.URL})
		}

		return err
	}

	// Determinate if content type is allowed
	allowed := false
	contentType := r.Headers()["Content-Type"]

	if allowedMimeTypes, err := state.configClient.GetAllowedMimeTypes(); err == nil {
		if len(allowedMimeTypes) == 0 {
			allowed = true
		}

		for _, allowedMimeType := range allowedMimeTypes {
			if strings.Contains(contentType, allowedMimeType.ContentType) {
				allowed = true
				break
			}
		}
	}

	if !allowed {
		return fmt.Errorf("%s (%s): %w", evt.URL, contentType, errContentTypeNotAllowed)
	}

	// Ready body
	b, err := ioutil.ReadAll(r.Body())
	if err != nil {
		return err
	}

	res := event.NewResourceEvent{
		URL:     evt.URL,
		Body:    string(b),
		Headers: r.Headers(),
		Time:    state.clock.Now(),
	}

	if err := subscriber.PublishEvent(&res); err != nil {
		return err
	}

	return nil
}
