package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Flush(ctx context.Context) error
	Close() error
}

type redisClient struct {
	client *redis.Client
}

func New(ctx context.Context) (Client, error) {
	addr := getEnvOrDefault("REDIS_ADDR", "localhost:6379")
	db, _ := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))
	password := os.Getenv("REDIS_PASSWORD")

	instance := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       db,
		Password: password,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := instance.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к Redis: %w", err)
	}

	return &redisClient{client: instance}, nil
}

func (c *redisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil && err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (c *redisClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *redisClient) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *redisClient) Flush(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

func (c *redisClient) Close() error {
	return c.client.Close()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

