package database

import (
	"github.com/creekorful/trandoshan/internal/indexer/client"
	"time"
)

//go:generate mockgen -destination=../database_mock/database_mock.go -package=database_mock . Database

// ResourceIdx represent a resource as stored in elasticsearch
type ResourceIdx struct {
	URL         string            `json:"url"`
	Body        string            `json:"body"`
	Time        time.Time         `json:"time"`
	Title       string            `json:"title"`
	Meta        map[string]string `json:"meta"`
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers"`
}

// Database is the interface used to abstract communication
// with the persistence unit
type Database interface {
	SearchResources(params *client.ResSearchParams) ([]ResourceIdx, error)
	CountResources(params *client.ResSearchParams) (int64, error)
	AddResource(res ResourceIdx) error
}
