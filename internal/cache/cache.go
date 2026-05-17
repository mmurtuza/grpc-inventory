// Package cache defines the Cache interface and shared key helpers.
package cache

import (
	"context"
	"time"
)

const (
	// SKUTTL is how long a single inventory item is cached in Redis.
	SKUTTL = 5 * time.Minute
	// LowStockTTL is how long the low-stock list is cached in Redis.
	LowStockTTL = 1 * time.Minute
)

// Cache is the interface used by REST API handlers.
// RedisCache implements it for production; NoopCache for testing/fallback.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Ping(ctx context.Context) error
}

// SKUKey returns the Redis cache key for a single inventory item.
func SKUKey(sku string) string { return "inv:sku:" + sku }

// LowStockKey returns the Redis cache key for the low-stock list.
func LowStockKey() string { return "inv:low-stock" }
