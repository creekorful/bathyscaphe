package api

import (
	idxclient "github.com/creekorful/trandoshan/internal/indexer/client"
	schclient "github.com/creekorful/trandoshan/internal/scheduler/client"
)

// Client is a client for the whole existing APIs
type Client interface {
	SearchResources(params *idxclient.ResSearchParams) ([]idxclient.ResourceDto, int64, error)
	ScheduleURL(url string) error
}

type client struct {
	indexerClient   idxclient.Client
	schedulerClient schclient.Client
}

// NewClient return a new Client instance
func NewClient(baseURL string) Client {
	return &client{
		indexerClient:   idxclient.NewClient(baseURL),
		schedulerClient: schclient.NewClient(baseURL),
	}
}

func (client *client) SearchResources(params *idxclient.ResSearchParams) ([]idxclient.ResourceDto, int64, error) {
	return client.indexerClient.SearchResources(params)
}

func (client *client) ScheduleURL(url string) error {
	return client.schedulerClient.ScheduleURL(url)
}
