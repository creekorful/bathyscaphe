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
	if err != nil && err != redis.Nil {
		return nil, err
	}

	return val, nil
}

func (rc *redisCache) SetBytes(key string, value []byte, TTL time.Duration) error {
	return rc.client.Set(context.Background(), rc.getKey(key), value, TTL).Err()
}

func (rc *redisCache) GetInt64(key string) (int64, error) {
	val, err := rc.client.Get(context.Background(), rc.getKey(key)).Int64()
	if err != nil && err != redis.Nil {
		return 0, err
	}

	return val, nil
}

func (rc *redisCache) SetInt64(key string, value int64, TTL time.Duration) error {
	return rc.client.Set(context.Background(), rc.getKey(key), value, TTL).Err()
}

func (rc *redisCache) GetManyInt64(keys []string) (map[string]int64, error) {
	pipeline := rc.client.Pipeline()

	// Execute commands and keep pointer to them
	commands := map[string]*redis.StringCmd{}
	for _, key := range keys {
		commands[key] = pipeline.Get(context.Background(), rc.getKey(key))
	}

	// Execute pipeline
	if _, err := pipeline.Exec(context.Background()); err != nil && err != redis.Nil {
		return nil, err
	}

	// Get back values
	values := map[string]int64{}
	for _, key := range keys {
		val, err := commands[key].Int64()
		if err != nil {
			// If it's a real error
			if err != redis.Nil {
				return nil, err
			}
		} else {
			// Only returns entry if there's one
			values[key] = val
		}
	}

	return values, nil
}

func (rc *redisCache) SetManyInt64(values map[string]int64, TTL time.Duration) error {
	pipeline := rc.client.TxPipeline()

	for key, value := range values {
		pipeline.Set(context.Background(), rc.getKey(key), value, TTL)
	}

	_, err := pipeline.Exec(context.Background())
	return err
}

func (rc *redisCache) getKey(key string) string {
	if rc.keyPrefix == "" {
		return key
	}

	return fmt.Sprintf("%s:%s", rc.keyPrefix, key)
}
