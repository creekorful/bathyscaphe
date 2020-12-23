package configapi

import (
	"github.com/creekorful/trandoshan/internal/configapi/api"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

type State struct {
	api api.ConfigAPI
}

func (s *State) getConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	b, err := s.api.Get(key)
	if err != nil {
		log.Printf("error while retrieving configuration: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (s *State) setConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error while reading body: %s", err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if err := s.api.Set(key, b); err != nil {
		log.Printf("error while setting configuration: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
