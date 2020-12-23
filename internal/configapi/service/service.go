package service

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/configapi/database"
	"github.com/creekorful/trandoshan/internal/event"
)

// Service is the functionality layer provided by the ConfigAPI
type Service struct {
	db  database.Database
	pub event.Publisher
}

// Get config using his key
func (s *Service) Get(key string) ([]byte, error) {
	return s.db.Get(key)
}

// Set config value
func (s *Service) Set(key string, value []byte) error {
	if err := s.db.Set(key, value); err != nil {
		return err
	}

	// publish event to notify config changed
	return s.pub.PublishJSON(fmt.Sprintf("config.%s", key), value)
}
