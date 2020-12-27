package crawler

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/clock"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/crawler/http"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"strings"
)

var errContentTypeNotAllowed = fmt.Errorf("content type is not allowed")

type State struct {
	httpClient   http.Client
	clock        clock.Clock
	configClient configapi.Client
}

func (state *State) Name() string {
	return "crawler"
}

func (state *State) Flags() []string {
	return []string{process.HubURIFlag, process.TorURIFlag, process.UserAgentFlag, process.ConfigAPIURIFlag}
}

func (state *State) Provide(provider process.Provider) error {
	httpClient, err := provider.FastHTTPClient()
	if err != nil {
		return err
	}
	state.httpClient = httpClient

	cl, err := provider.Clock()
	if err != nil {
		return err
	}
	state.clock = cl

	configClient, err := provider.ConfigClient([]string{configapi.AllowedMimeTypesKey})
	if err != nil {
		return err
	}
	state.configClient = configClient

	return nil
}

func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.NewURLExchange, Queue: "crawlingQueue", Handler: state.handleNewURLEvent},
	}
}

func (state *State) handleNewURLEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewURLEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	log.Debug().Str("url", evt.URL).Msg("Processing URL")

	r, err := state.httpClient.Get(evt.URL)
	if err != nil {
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
