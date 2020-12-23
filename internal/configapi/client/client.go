package client

import (
	"github.com/creekorful/trandoshan/internal/event"
	"time"
)

//go:generate mockgen -destination=../client_mock/client_mock.go -package=client_mock . Client

const (
	// ForbiddenMimeTypesKey is the key to access the forbidden mime types config
	ForbiddenMimeTypesKey = "forbidden-mime-types"
	// ForbiddenHostnamesKey is the key to access the forbidden hostnames config
	ForbiddenHostnamesKey = "forbidden-hostnames"
	// RefreshDelayKey is the key to access the refresh delay config
	RefreshDelayKey = "refresh-delay"
)

// ForbiddenMimeType is the mime types who's crawling is forbidden
type ForbiddenMimeType struct {
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
	GetForbiddenMimeTypes() ([]ForbiddenMimeType, error)
	SetForbiddenMimeTypes(values []ForbiddenMimeType) error

	GetForbiddenHostnames() ([]ForbiddenHostname, error)
	SetForbiddenHostnames(values []ForbiddenHostname) error

	GetRefreshDelay() (RefreshDelay, error)
	SetRefreshDelay(value RefreshDelay) error
}

type client struct {
}

// NewConfigClient create a new client for the ConfigAPI
func NewConfigClient(configAPIURL string, subscriber event.Subscriber, keys []string) (Client, error) {
	return nil, nil // TODO
}
