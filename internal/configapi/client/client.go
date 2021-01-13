package client

//go:generate mockgen -destination=../client_mock/client_mock.go -package=client_mock . Client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

const (
	// AllowedMimeTypesKey is the key to access the allowed mime types config
	AllowedMimeTypesKey = "allowed-mime-types"
	// ForbiddenHostnamesKey is the key to access the forbidden hostnames config
	ForbiddenHostnamesKey = "forbidden-hostnames"
	// RefreshDelayKey is the key to access the refresh delay config
	RefreshDelayKey = "refresh-delay"
	// BlackListConfigKey is the key to access the blacklist configuration
	BlackListConfigKey = "blacklist-config"
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

// BlackListConfig is the config used for hostname blacklisting
type BlackListConfig struct {
	Threshold int64         `json:"threshold"`
	TTL       time.Duration `json:"ttl"`
}

// Client is a nice client interface for the ConfigAPI
type Client interface {
	GetAllowedMimeTypes() ([]MimeType, error)
	GetForbiddenHostnames() ([]ForbiddenHostname, error)
	GetRefreshDelay() (RefreshDelay, error)
	GetBlackListConfig() (BlackListConfig, error)

	Set(key string, value interface{}) error
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
	blackListConfig    BlackListConfig
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

func (c *client) GetAllowedMimeTypes() ([]MimeType, error) {
	c.mutexes[AllowedMimeTypesKey].RLock()
	defer c.mutexes[AllowedMimeTypesKey].RUnlock()

	return c.allowedMimeTypes, nil
}

func (c *client) setAllowedMimeTypes(values []MimeType) error {
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

func (c *client) setForbiddenHostnames(values []ForbiddenHostname) error {
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

func (c *client) setRefreshDelay(value RefreshDelay) error {
	c.mutexes[RefreshDelayKey].Lock()
	defer c.mutexes[RefreshDelayKey].Unlock()

	c.refreshDelay = value

	return nil
}

func (c *client) GetBlackListConfig() (BlackListConfig, error) {
	c.mutexes[BlackListConfigKey].RLock()
	defer c.mutexes[BlackListConfigKey].RUnlock()

	return c.blackListConfig, nil
}

func (c *client) setBlackListConfig(value BlackListConfig) error {
	c.mutexes[BlackListConfigKey].Lock()
	defer c.mutexes[BlackListConfigKey].Unlock()

	c.blackListConfig = BlackListConfig{
		Threshold: value.Threshold,
		TTL:       value.TTL * time.Second, // TTL is in seconds
	}

	return nil
}

func (c *client) Set(key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/config/%s", c.configAPIURL, key), bytes.NewReader(b))
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

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
	case AllowedMimeTypesKey:
		var val []MimeType
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.setAllowedMimeTypes(val); err != nil {
			return err
		}
		break
	case ForbiddenHostnamesKey:
		var val []ForbiddenHostname
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.setForbiddenHostnames(val); err != nil {
			return err
		}
		break
	case RefreshDelayKey:
		var val RefreshDelay
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.setRefreshDelay(val); err != nil {
			return err
		}
		break
	case BlackListConfigKey:
		var val BlackListConfig
		if err := json.Unmarshal(value, &val); err != nil {
			return err
		}
		if err := c.setBlackListConfig(val); err != nil {
			return err
		}
		break
	default:
		return fmt.Errorf("non managed value type: %s", key)
	}

	log.Trace().Str("key", key).Bytes("value", value).Msg("Successfully set value")

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
