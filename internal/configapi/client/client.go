package client

import (
	"encoding/json"
	"fmt"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

//go:generate mockgen -destination=../client_mock/client_mock.go -package=client_mock . Client

const (
	// ForbiddenMimeTypesKey is the key to access the forbidden mime types config
	ForbiddenMimeTypesKey = "forbidden-mime-types"
	// AllowedMimeTypesKey is the key to access the allowed mime types config
	AllowedMimeTypesKey = "allowed-mime-types"
	// ForbiddenHostnamesKey is the key to access the forbidden hostnames config
	ForbiddenHostnamesKey = "forbidden-hostnames"
	// RefreshDelayKey is the key to access the refresh delay config
	RefreshDelayKey = "refresh-delay"
)

// MimeType is the mime type as represented in the config
type MimeType struct {
	// The content-type
	ContentType string `json:"content-type"`
	// The list of associated extensions
	Extensions []string `json:"extensions"`
}

// ForbiddenHostname is the hostnames who's crawling is forbidden
type ForbiddenHostname struct {
	Hostname string `json:"hostname"`
}

// RefreshDelay is the refresh delay for re-crawling
type RefreshDelay struct {
	Delay time.Duration `json:"delay"`
}

// Client is a nice client interface for the ConfigAPI
type Client interface {
	GetForbiddenMimeTypes() ([]MimeType, error)
	SetForbiddenMimeTypes(values []MimeType) error

	GetAllowedMimeTypes() ([]MimeType, error)
	SetAllowedMimeTypes(values []MimeType) error

	GetForbiddenHostnames() ([]ForbiddenHostname, error)
	SetForbiddenHostnames(values []ForbiddenHostname) error

	GetRefreshDelay() (RefreshDelay, error)
	SetRefreshDelay(value RefreshDelay) error
}

type client struct {
	configAPIURL string
	sub          event.Subscriber
	mutexes      map[string]*sync.RWMutex
	keys         []string

	forbiddenMimeTypes []MimeType
	allowedMimeTypes   []MimeType
	forbiddenHostnames []ForbiddenHostname
	refreshDelay       RefreshDelay
}

// NewConfigClient create a new client for the ConfigAPI.
func NewConfigClient(configAPIURL string, subscriber event.Subscriber, keys []string) (Client, error) {
	client := &client{
		configAPIURL: configAPIURL,
		sub:          subscriber,
		mutexes:      map[string]*sync.RWMutex{},
		keys:         keys,
	}

	// Pre-load wanted keys & create mutex
	for _, key := range keys {
		client.mutexes[key] = &sync.RWMutex{}

		val, err := client.get(key)
		if err != nil {
			return nil, err
		}

		if err := client.setValue(key, val); err != nil {
			return nil, err
		}
	}

	// Subscribe for config changed
	if err := client.sub.SubscribeAll(event.ConfigExchange, client.handleConfigEvent); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *client) GetForbiddenMimeTypes() ([]MimeType, error) {
	c.mutexes[ForbiddenMimeTypesKey].RLock()
	defer c.mutexes[ForbiddenMimeTypesKey].RUnlock()

	return c.forbiddenMimeTypes, nil
}

func (c *client) SetForbiddenMimeTypes(values []MimeType) error {
	c.mutexes[ForbiddenMimeTypesKey].Lock()
	defer c.mutexes[ForbiddenMimeTypesKey].Unlock()

	c.forbiddenMimeTypes = values

	return nil
}

func (c *client) GetAllowedMimeTypes() ([]MimeType, error) {
	c.mutexes[AllowedMimeTypesKey].RLock()
	defer c.mutexes[AllowedMimeTypesKey].RUnlock()

	return c.allowedMimeTypes, nil
}

func (c *client) SetAllowedMimeTypes(values []MimeType) error {
	c.mutexes[AllowedMimeTypesKey].Lock()
	defer c.mutexes[AllowedMimeTypesKey].Unlock()

	c.allowedMimeTypes = values

	return nil
}

func (c *client) GetForbiddenHostnames() ([]ForbiddenHostname, error) {
	c.mutexes[ForbiddenHostnamesKey].RLock()
	defer c.mutexes[ForbiddenHostnamesKey].RUnlock()

	return c.forbiddenHostnames, nil
}

func (c *client) SetForbiddenHostnames(values []ForbiddenHostname) error {
	c.mutexes[ForbiddenHostnamesKey].Lock()
	defer c.mutexes[ForbiddenHostnamesKey].Unlock()

	c.forbiddenHostnames = values

	return nil
}

func (c *client) GetRefreshDelay() (RefreshDelay, error) {
	c.mutexes[RefreshDelayKey].RLock()
	defer c.mutexes[RefreshDelayKey].RUnlock()

	return c.refreshDelay, nil
}

func (c *client) SetRefreshDelay(value RefreshDelay) error {
	c.mutexes[RefreshDelayKey].Lock()
	defer c.mutexes[RefreshDelayKey].Unlock()

	c.refreshDelay = value

	return nil
}

func (c *client) get(key string) ([]byte, error) {
	r, err := http.Get(fmt.Sprintf("%s/config/%s", c.configAPIURL, key))
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (c *client) setValue(key string, value []byte) error {
	switch key {
	case ForbiddenMimeTypesKey:
		var val []MimeType
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.SetForbiddenMimeTypes(val); err != nil {
			return err
		}
		break
	case AllowedMimeTypesKey:
		var val []MimeType
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.SetAllowedMimeTypes(val); err != nil {
			return err
		}
		break
	case ForbiddenHostnamesKey:
		var val []ForbiddenHostname
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.SetForbiddenHostnames(val); err != nil {
			return err
		}
		break
	case RefreshDelayKey:
		var val RefreshDelay
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.SetRefreshDelay(val); err != nil {
			return err
		}
		break
	default:
		return fmt.Errorf("non managed value type: %s", key)
	}

	log.Debug().Str("key", key).Bytes("value", value).Msg("Successfully set value")

	return nil
}

func (c *client) handleConfigEvent(_ event.Subscriber, msg event.RawMessage) error {
	// Make sure we have the header
	configKey, ok := msg.Headers["Config-Key"].(string)
	if !ok {
		return fmt.Errorf("message has no Config-Key header")
	}

	for _, key := range c.keys {
		if key == configKey {
			if err := c.setValue(configKey, msg.Body); err != nil {
				return err
			}
			break
		}
	}

	return nil // TODO
}
