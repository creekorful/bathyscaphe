package cache

//go:generate mockgen -destination=../cache_mock/cache_mock.go -package=cache_mock . Cache

import (
	"time"
)

var (
	// NoTTL define an entry that lives forever
	NoTTL = time.Duration(0)
)

// Cache represent a KV database
type Cache interface {
	GetBytes(key string) ([]byte, error)
	SetBytes(key string, value []byte, TTL time.Duration) error

	GetInt64(key string) (int64, error)
	SetInt64(key string, value int64, TTL time.Duration) error

	GetManyInt64(keys []string) (map[string]int64, error)
	SetManyInt64(values map[string]int64, TTL time.Duration) error

	Remove(key string) error
}
