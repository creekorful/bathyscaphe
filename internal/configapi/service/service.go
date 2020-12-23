package service

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/configapi/api"
	"github.com/creekorful/trandoshan/internal/configapi/database"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/rs/zerolog/log"
)

type service struct {
	db  database.Database
	pub event.Publisher
}

// NewService create a new service exposing the ConfigAPI features
func NewService(db database.Database, pub event.Publisher) (api.ConfigAPI, error) {
	return &service{
		db:  db,
		pub: pub,
	}, nil
}

// Get config using his key
func (s *service) Get(key string) ([]byte, error) {
	log.Debug().Str("key", key).Msg("Getting key")
	return s.db.Get(key)
}

// Set config value
func (s *service) Set(key string, value []byte) error {
	log.Debug().Str("key", key).Bytes("value", value).Msg("Setting key")

	if err := s.db.Set(key, value); err != nil {
		return err
	}

	// publish event to notify config changed
	return s.pub.PublishJSON(fmt.Sprintf("config.%s", key), value)
}
