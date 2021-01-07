package index

//go:generate mockgen -destination=../index_mock/index_mock.go -package=index_mock . Index

import (
	"fmt"
	"time"
)

const (
	// Elastic is an Index backed by ES instance
	Elastic = "elastic"
	// Local is an Index backed by local FS instance
	Local = "local"
)

// Resource represent a resource that should be indexed
type Resource struct {
	URL     string
	Time    time.Time
	Body    string
	Headers map[string]string
}

// Index is the interface used to abstract communication with the persistence unit
type Index interface {
	IndexResource(resource Resource) error
	IndexResources(resources []Resource) error
}

// NewIndex create a new index using given driver, destination
func NewIndex(driver string, dest string) (Index, error) {
	switch driver {
	case Elastic:
		return newElasticIndex(dest)
	case Local:
		return newLocalIndex(dest)
	default:
		return nil, fmt.Errorf("no driver named %s found", driver)
	}
}
