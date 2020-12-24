package database

import (
	"context"
	"github.com/go-redis/redis/v8"
)

type redisDatabase struct {
	rc *redis.Client
}

// NewRedisDatabase returns a new database that use Redis as backend
func NewRedisDatabase(uri string) (Database, error) {
	return &redisDatabase{
		rc: redis.NewClient(&redis.Options{
			Addr:     uri,
			Password: "", // no password set
			DB:       0,  // use default DB
		}),
	}, nil
}

func (rd *redisDatabase) Get(key string) ([]byte, error) {
	return rd.rc.Get(context.Background(), key).Bytes()
}

func (rd *redisDatabase) Set(key string, value []byte) error {
	return rd.rc.Set(context.Background(), key, value, 0).Err()
}
