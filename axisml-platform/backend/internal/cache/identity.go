package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
)

// IdentityCache fronts the PostgreSQL identity load (auth.IdentityStore) with
// Redis. LoadIdentity runs on every authenticated request and costs two queries
// (user row + RBAC bindings), so caching it removes the bulk of the auth-path
// read load.
//
// Identity caching is security-relevant: a stale entry could keep a removed
// member or disabled account authorized. Two mechanisms bound staleness — an
// explicit Invalidate at every binding/account mutation, plus a short backstop
// TTL in case a bust is ever missed. Errors (disabled / not-found) are never
// cached.
type IdentityCache struct {
	base auth.IdentityStore
	c    Cache
	ttl  time.Duration
	log  *slog.Logger
}

// NewIdentityCache wraps base with a Redis-backed identity cache.
func NewIdentityCache(base auth.IdentityStore, c Cache, ttl time.Duration, log *slog.Logger) *IdentityCache {
	return &IdentityCache{base: base, c: c, ttl: ttl, log: log}
}

func identityKey(userID string) string { return "platform:identity:" + userID }

// LoadIdentity returns the cached identity on a hit, otherwise loads from
// PostgreSQL and caches the result.
func (i *IdentityCache) LoadIdentity(ctx context.Context, userID string) (*auth.Identity, error) {
	if b, ok, err := i.c.Get(ctx, identityKey(userID)); err != nil {
		i.log.Debug("identity cache: get failed, falling back to PostgreSQL", "error", err)
	} else if ok {
		var id auth.Identity
		if err := json.Unmarshal(b, &id); err == nil {
			return &id, nil
		}
		i.log.Debug("identity cache: unmarshal failed, reloading", "error", err)
	}
	id, err := i.base.LoadIdentity(ctx, userID)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(id); err == nil {
		if err := i.c.Set(ctx, identityKey(userID), b, i.ttl); err != nil {
			i.log.Debug("identity cache: set failed", "error", err)
		}
	}
	return id, nil
}

// Invalidate drops a user's cached identity. Call it after any change to the
// user's account state or tenant bindings. Safe to call when the cache is noop.
func (i *IdentityCache) Invalidate(ctx context.Context, userID string) {
	if err := i.c.Del(ctx, identityKey(userID)); err != nil {
		i.log.Debug("identity cache: invalidate failed", "error", err)
	}
}
