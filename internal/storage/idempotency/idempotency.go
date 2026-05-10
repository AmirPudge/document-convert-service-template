package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	rdb *redis.Client
}

const TTL = 24 * time.Hour

func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisClient{rdb: rdb}, nil
}

func (r *RedisClient) CheckKey(ctx context.Context, requestID string) (bool, error) {
	return r.rdb.SetNX(ctx, requestID, "AMIR", TTL).Result()
}

func (r *RedisClient) DeleteKey(ctx context.Context, requestID string) error {
	return r.rdb.Del(ctx, requestID).Err()
}

func (c *RedisClient) Close() error {
	return c.rdb.Close()
}
