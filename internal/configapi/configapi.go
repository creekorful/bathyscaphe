package configapi

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/configapi/database"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// GetApp return the config api app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-configapi",
		Version: "0.7.0",
		Usage:   "Trandoshan ConfigAPI component",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			util.GetHubURI(),
			&cli.StringFlag{
				Name:     "db-uri",
				Usage:    "URI to the database server",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:  "default-value",
				Usage: "Set default value of key. (format key=value)",
			},
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)

	log.Info().
		Str("ver", ctx.App.Version).
		Str("hub-uri", ctx.String("hub-uri")).
		Str("db-uri", ctx.String("db-uri")).
		Msg("Starting tdsh-configapi")

	// Create publisher
	pub, err := event.NewPublisher(ctx.String("hub-uri"))
	if err != nil {
		return err
	}

	// Create database connection
	db, err := database.NewRedisDatabase(ctx.String("db-uri"))
	if err != nil {
		return err
	}

	// Parse default values
	defaultValues := map[string]string{}
	for _, value := range ctx.StringSlice("default-value") {
		parts := strings.Split(value, "=")

		if len(parts) == 2 {
			defaultValues[parts[0]] = parts[1]
		}
	}

	// Set default values if needed
	if len(defaultValues) > 0 {
		if err := setDefaultValues(db, defaultValues); err != nil {
			log.Err(err).Msg("error while setting default values")
			return err
		}
	}

	state := state{
		db:  db,
		pub: pub,
	}

	r := mux.NewRouter()
	r.HandleFunc("/config/{key}", state.getConfiguration).Methods(http.MethodGet)
	r.HandleFunc("/config/{key}", state.setConfiguration).Methods(http.MethodPut)

	srv := &http.Server{
		Addr: "0.0.0.0:8080",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	return srv.ListenAndServe()
}

type state struct {
	db  database.Database
	pub event.Publisher
}

func (state *state) getConfiguration(w http.ResponseWriter, r *http.Request) {
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

func (state *state) setConfiguration(w http.ResponseWriter, r *http.Request) {
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
