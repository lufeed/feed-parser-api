package cache

import (
	"context"
	"github.com/lufeed/feed-parser-api/internal/config"
	"github.com/lufeed/feed-parser-api/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

var ctx = context.Background()

var client *redis.Client

func Initialize(cfg *config.AppConfig) error {
	client = redis.NewClient(&redis.Options{
		Addr:     cfg.Cache.Address,
		Password: cfg.Cache.Pass,
		DB:       0,
	})

	// Test the connection to Redis
	_, err := client.Ping(ctx).Result()
	if err != nil {
		logger.GetLogger().With(zap.Error(err)).Fatal("Failed to connect to Redis")
		return err
	}

	logger.GetLogger().Info("Connected to cache")
	return nil
}

// SetCache sets a value in Redis cache with an expiration time
func SetCache(key string, value interface{}, expiration time.Duration) error {
	err := client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

// GetCache retrieves a value from Redis cache
func GetCache(key string) (string, error) {
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func DeleteCache(key string) error {
	return client.Del(ctx, key).Err()
}

func Publish(key string, data []byte) error {
	return client.Publish(ctx, key, data).Err()
}

func Subscribe(key string) *redis.PubSub {
	return client.Subscribe(ctx, key)
}
