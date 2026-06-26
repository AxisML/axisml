// Package cache is the platform-backend's optional Redis accelerator for the
// authentication hot path. Every authenticated request would otherwise make two
// to three PostgreSQL round-trips before the handler runs — a session-validity
// lookup and an identity/RBAC load. This package fronts both with Redis.
//
// Redis is strictly an accelerator, never a source of truth: PostgreSQL remains
// authoritative. An empty address yields a noopCache (identical behavior to no
// cache at all), and a transient Redis error falls back to the durable store
// per-operation, so the request path never fails because Redis is unhealthy.
package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/axisml/axisml/components/platform/internal/config"
)

// Cache is the minimal key/value surface the auth caches need. Implementations
// must treat every error as a miss-equivalent at the call site: callers fall
// back to PostgreSQL, so a Cache method returning an error is advisory only.
type Cache interface {
	// Get returns the value and true on a hit, nil and false on a miss.
	Get(ctx context.Context, key string) ([]byte, bool, error)
	// Set stores a value with a TTL. A non-positive ttl persists with no expiry.
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	// Del removes keys (no error on missing keys).
	Del(ctx context.Context, keys ...string) error
	// Enabled reports whether this is a real backing store (false for noop).
	Enabled() bool
	// Close releases the underlying client.
	Close() error
}

// New builds a Cache from config. When cfg.RedisAddr is empty it returns a
// noopCache. Otherwise it dials Redis and pings once: a failed ping is logged
// but does NOT fail construction, since go-redis reconnects transparently and
// the caches degrade to PostgreSQL until Redis recovers.
func New(cfg config.Config, log *slog.Logger) Cache {
	if cfg.Cache.Addr == "" {
		log.Info("cache disabled: cache.addr is unset, auth lookups go straight to PostgreSQL")
		return noopCache{}
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Cache.Addr,
		Password: cfg.Cache.Password,
		DB:       cfg.Cache.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Warn("cache: initial Redis ping failed, degrading to PostgreSQL until it recovers",
			"addr", cfg.Cache.Addr, "error", err)
	} else {
		log.Info("cache enabled", "addr", cfg.Cache.Addr)
	}
	return &redisCache{client: client, log: log}
}

// redisCache is the live Redis-backed implementation.
type redisCache struct {
	client *redis.Client
	log    *slog.Logger
}

func (r *redisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

func (r *redisCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if ttl < 0 {
		ttl = 0
	}
	return r.client.Set(ctx, key, val, ttl).Err()
}

func (r *redisCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *redisCache) Enabled() bool { return true }

func (r *redisCache) Close() error { return r.client.Close() }

// noopCache is the disabled implementation: every read misses and every write
// is dropped, so callers behave exactly as they would with no cache.
type noopCache struct{}

func (noopCache) Get(context.Context, string) ([]byte, bool, error)        { return nil, false, nil }
func (noopCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (noopCache) Del(context.Context, ...string) error                     { return nil }
func (noopCache) Enabled() bool                                            { return false }
func (noopCache) Close() error                                             { return nil }
