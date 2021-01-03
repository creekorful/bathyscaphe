package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type redisCache struct {
	client *redis.Client
}

// NewRedisCache return a new Cache using redis as backend
func NewRedisCache(URI string) (Cache, error) {
	return &redisCache{
		client: redis.NewClient(&redis.Options{
			Addr: URI,
			DB:   0,
		}),
	}, nil
}

func (rc *redisCache) GetBytes(key string) ([]byte, error) {
	val, err := rc.client.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		err = ErrNIL
	}

	return val, err
}

func (rc *redisCache) SetBytes(key string, value []byte, TTL time.Duration) error {
	return rc.client.Set(context.Background(), key, value, TTL).Err()
}

func (rc *redisCache) GetInt64(key string) (int64, error) {
	val, err := rc.client.Get(context.Background(), key).Int64()
	if err == redis.Nil {
		err = ErrNIL
	}

	return val, err
}

func (rc *redisCache) SetInt64(key string, value int64, TTL time.Duration) error {
	return rc.client.Set(context.Background(), key, value, TTL).Err()
}
