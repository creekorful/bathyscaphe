package configapi

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/cache"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/process"
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

// CommonFlags return process common flags
func (state *State) CommonFlags() []string {
	return []string{process.HubURIFlag}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "db-uri",
			Usage:    "URI to the database server",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:  "default-value",
			Usage: "Set default value of key. (format key=value)",
		},
	}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	// TODO init cache

	pub, err := provider.Publisher()
	if err != nil {
		return err
	}
	state.pub = pub

	defaultValues := map[string]string{}
	for _, value := range provider.GetValues("default-value") {
		parts := strings.Split(value, "=")

		if len(parts) == 2 {
			defaultValues[parts[0]] = parts[1]
		}
	}
	if len(defaultValues) > 0 {
		if err := setDefaultValues(nil, defaultValues); err != nil {
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
func (state *State) HTTPHandler(provider process.Provider) http.Handler {
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
		if _, err := configCache.GetBytes(key); err == cache.ErrNIL {
			if err := configCache.SetBytes(key, []byte(value), cache.NoTTL); err != nil {
				return fmt.Errorf("error while setting default value of %s: %s", key, err)
			}
		}
	}

	return nil
}
