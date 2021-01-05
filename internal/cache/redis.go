package cache

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

type redisCache struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisCache return a new Cache using redis as backend
func NewRedisCache(URI string, keyPrefix string) (Cache, error) {
	return &redisCache{
		client: redis.NewClient(&redis.Options{
			Addr: URI,
			DB:   0,
		}),
		keyPrefix: keyPrefix,
	}, nil
}

func (rc *redisCache) GetBytes(key string) ([]byte, error) {
	val, err := rc.client.Get(context.Background(), rc.getKey(key)).Bytes()
	if err == redis.Nil {
		err = ErrNIL
	}

	return val, err
}

func (rc *redisCache) SetBytes(key string, value []byte, TTL time.Duration) error {
	return rc.client.Set(context.Background(), rc.getKey(key), value, TTL).Err()
}

func (rc *redisCache) GetInt64(key string) (int64, error) {
	val, err := rc.client.Get(context.Background(), rc.getKey(key)).Int64()
	if err == redis.Nil {
		err = ErrNIL
	}

	return val, err
}

func (rc *redisCache) SetInt64(key string, value int64, TTL time.Duration) error {
	return rc.client.Set(context.Background(), rc.getKey(key), value, TTL).Err()
}

func (rc *redisCache) getKey(key string) string {
	if rc.keyPrefix == "" {
		return key
	}

	return fmt.Sprintf("%s:%s", rc.keyPrefix, key)
}
