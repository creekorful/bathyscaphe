package configapi

import (
	"fmt"
	"github.com/darkspot-org/bathyscaphe/internal/cache"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
	"strings"
)

// State represent the application state
type State struct {
	configCache cache.Cache
	pub         event.Publisher
}

// Name return the process name
func (state *State) Name() string {
	return "configapi"
}

// Description return the process description
func (state *State) Description() string {
	return `
The ConfigAPI component. It serves as a centralized K/V database
with notification support.
This component expose a REST API to allow other process to retrieve
configuration as startup time, and to allow value update at runtime.
Each time a configuration is update trough the API, an event will
be dispatched so that running processes can update their local values.

This component produces the 'config' event.`
}

// Features return the process features
func (state *State) Features() []process.Feature {
	return []process.Feature{process.EventFeature, process.CacheFeature}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "default-value",
			Usage: "Set default value of key. (format key=value)",
		},
	}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	configCache, err := provider.Cache("configuration")
	if err != nil {
		return err
	}
	state.configCache = configCache

	pub, err := provider.Publisher()
	if err != nil {
		return err
	}
	state.pub = pub

	defaultValues := map[string]string{}
	for _, value := range provider.GetStrValues("default-value") {
		parts := strings.Split(value, "=")

		if len(parts) == 2 {
			defaultValues[parts[0]] = parts[1]
		}
	}
	if len(defaultValues) > 0 {
		if err := setDefaultValues(configCache, defaultValues); err != nil {
			return err
		}
	}

	return nil // TODO
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/config/{key}", state.getConfiguration).Methods(http.MethodGet)
	r.HandleFunc("/config/{key}", state.setConfiguration).Methods(http.MethodPut)

	return r
}

func (state *State) getConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	log.Debug().Str("key", key).Msg("Getting key")

	b, err := state.configCache.GetBytes(key)
	if err != nil {
		log.Err(err).Msg("error while retrieving configuration")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (state *State) setConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Err(err).Msg("error while reading body")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	log.Debug().Str("key", key).Bytes("value", b).Msg("Setting key")

	if err := state.configCache.SetBytes(key, b, cache.NoTTL); err != nil {
		log.Err(err).Msg("error while setting configuration")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// publish event to notify config changed
	if err := state.pub.PublishJSON(event.ConfigExchange, event.RawMessage{
		Body:    b,
		Headers: map[string]interface{}{"Config-Key": key},
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func setDefaultValues(configCache cache.Cache, values map[string]string) error {
	for key, value := range values {
		b, err := configCache.GetBytes(key)
		if err != nil {
			return err
		}

		if b == nil {
			if err := configCache.SetBytes(key, []byte(value), cache.NoTTL); err != nil {
				return fmt.Errorf("error while setting default value of %s: %s", key, err)
			}
		}
	}

	return nil
}
