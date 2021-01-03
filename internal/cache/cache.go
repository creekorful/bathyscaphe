package cache

import "time"

//go:generate mockgen -destination=../cache_mock/cache_mock.go -package=cache_mock . Cache

// Cache represent a KV database
type Cache interface {
	GetBytes(key string) ([]byte, error)
	SetBytes(key string, value []byte, TTL time.Duration) error

	GetInt64(key string) (int64, error)
	SetInt64(key string, value int64, TTL time.Duration) error
}
