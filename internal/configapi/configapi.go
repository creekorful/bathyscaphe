package configapi

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/configapi/database"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
	"strings"
)

// State represent the application state
type State struct {
	db  database.Database
	pub event.Publisher
}

// Name return the process name
func (state *State) Name() string {
	return "configapi"
}

// CommonFlags return process common flags
func (state *State) CommonFlags() []string {
	return []string{}
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
	db, err := database.NewRedisDatabase(provider.GetValue("db-uri"))
	if err != nil {
		return err
	}
	state.db = db

	defaultValues := map[string]string{}
	for _, value := range provider.GetValues("default-value") {
		parts := strings.Split(value, "=")

		if len(parts) == 2 {
			defaultValues[parts[0]] = parts[1]
		}
	}
	if len(defaultValues) > 0 {
		if err := setDefaultValues(db, defaultValues); err != nil {
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

	b, err := state.db.Get(key)
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

	if err := state.db.Set(key, b); err != nil {
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

func setDefaultValues(service database.Database, values map[string]string) error {
	for key, value := range values {
		if _, err := service.Get(key); err == redis.Nil {
			if err := service.Set(key, []byte(value)); err != nil {
				return fmt.Errorf("error while setting default value of %s: %s", key, err)
			}
		}
	}

	return nil
}
