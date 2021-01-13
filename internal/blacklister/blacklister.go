package blacklister

import (
	"fmt"
	"github.com/darkspot-org/bathyscaphe/internal/cache"
	configapi "github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	chttp "github.com/darkspot-org/bathyscaphe/internal/http"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
)

var errAlreadyBlacklisted = fmt.Errorf("hostname is already blacklisted")

// State represent the application state
type State struct {
	configClient  configapi.Client
	hostnameCache cache.Cache
	httpClient    chttp.Client
}

// Name return the process name
func (state *State) Name() string {
	return "blacklister"
}

// Description return the process description
func (state *State) Description() string {
	return `
The blacklisting component. It consumes timeout URL event and will try to
crawl the hostname index page to determinate if the whole hostname does not
respond. If the hostname does not respond after a retry policy, it will
be blacklisted by the process and further crawling event involving the hostname
will be discarded by the crawling process. This allow us to not waste time
crawling for nothing.

This process consumes the 'url.timeout' event.`
}

// Features return the process features
func (state *State) Features() []process.Feature {
	return []process.Feature{process.EventFeature, process.ConfigFeature, process.CacheFeature, process.CrawlingFeature}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	hostnameCache, err := provider.Cache("down-hostname")
	if err != nil {
		return err
	}
	state.hostnameCache = hostnameCache

	configClient, err := provider.ConfigClient([]string{configapi.ForbiddenHostnamesKey, configapi.BlackListConfigKey})
	if err != nil {
		return err
	}
	state.configClient = configClient

	httpClient, err := provider.HTTPClient()
	if err != nil {
		return err
	}
	state.httpClient = httpClient

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.TimeoutURLExchange, Queue: "blacklistingQueue", Handler: state.handleTimeoutURLEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler() http.Handler {
	return nil
}

func (state *State) handleTimeoutURLEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.TimeoutURLEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	u, err := url.Parse(evt.URL)
	if err != nil {
		return err
	}

	// Make sure hostname is not already 'blacklisted'
	forbiddenHostnames, err := state.configClient.GetForbiddenHostnames()
	if err != nil {
		return err
	}

	// prevent duplicates
	found := false
	for _, hostname := range forbiddenHostnames {
		if hostname.Hostname == u.Hostname() {
			found = true
			break
		}
	}

	if found {
		return fmt.Errorf("%s %w", u.Hostname(), errAlreadyBlacklisted)
	}

	// Check by ourselves if the hostname doesn't respond
	_, err = state.httpClient.Get(fmt.Sprintf("%s://%s", u.Scheme, u.Host))
	if err != nil && err != chttp.ErrTimeout {
		return err
	}

	cacheKey := u.Hostname()

	if err == nil {
		log.Debug().
			Str("hostname", u.Hostname()).
			Msg("Response received.")

		// Host is not down, remove it from cache
		if err := state.hostnameCache.Remove(cacheKey); err != nil {
			return err
		}

		return nil
	}

	log.Debug().
		Str("hostname", u.Hostname()).
		Msg("Timeout confirmed")

	blackListConfig, err := state.configClient.GetBlackListConfig()
	if err != nil {
		return err
	}

	count, err := state.hostnameCache.GetInt64(cacheKey)
	if err != nil {
		return err
	}
	count++

	if count >= blackListConfig.Threshold {
		forbiddenHostnames, err := state.configClient.GetForbiddenHostnames()
		if err != nil {
			return err
		}

		// prevent duplicates
		found := false
		for _, hostname := range forbiddenHostnames {
			if hostname.Hostname == u.Hostname() {
				found = true
				break
			}
		}

		if found {
			log.Trace().Str("hostname", u.Hostname()).Msg("Skipping duplicate hostname")
		} else {
			log.Info().
				Str("hostname", u.Hostname()).
				Int64("count", count).
				Msg("Blacklisting hostname")

			forbiddenHostnames = append(forbiddenHostnames, configapi.ForbiddenHostname{Hostname: u.Hostname()})
			if err := state.configClient.Set(configapi.ForbiddenHostnamesKey, forbiddenHostnames); err != nil {
				return err
			}
		}
	}

	// Update count
	if err := state.hostnameCache.SetInt64(cacheKey, count, blackListConfig.TTL); err != nil {
		return err
	}

	return nil
}
