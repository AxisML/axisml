package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/axisml/axisml/components/platform/internal/auth"
)

// SessionCache fronts the PostgreSQL session allowlist (auth.SessionStore) with
// Redis. The check in middleware is an allowlist — a token is valid only while
// its session row exists and is unrevoked — so the cache stores only positive
// (active) entries: a hit means active, a miss falls through to PostgreSQL.
// Revoked or absent sessions are never cached, so a logout can never be masked
// by a stale "active" entry beyond the short backstop TTL.
type SessionCache struct {
	base auth.SessionStore
	c    Cache
	ttl  time.Duration
	log  *slog.Logger
}

// NewSessionCache wraps base with a Redis-backed positive cache. ttl is a short
// backstop (the entry is also dropped on Revoke); it bounds how long a missed
// revocation Del could leave a token cached as active.
func NewSessionCache(base auth.SessionStore, c Cache, ttl time.Duration, log *slog.Logger) *SessionCache {
	return &SessionCache{base: base, c: c, ttl: ttl, log: log}
}

func sessionKey(jti string) string { return "platform:sess:" + jti }

// Create records the session in PostgreSQL and write-throughs the active entry.
func (s *SessionCache) Create(ctx context.Context, jti, userID string, expiresAtUnix int64) error {
	if err := s.base.Create(ctx, jti, userID, expiresAtUnix); err != nil {
		return err
	}
	if err := s.c.Set(ctx, sessionKey(jti), []byte{'1'}, s.ttl); err != nil {
		s.log.Debug("session cache: set failed", "error", err)
	}
	return nil
}

// IsActive checks Redis first, falling back to PostgreSQL on a miss or error and
// repopulating the cache when the session is active.
func (s *SessionCache) IsActive(ctx context.Context, jti string) (bool, error) {
	if _, ok, err := s.c.Get(ctx, sessionKey(jti)); err != nil {
		s.log.Debug("session cache: get failed, falling back to PostgreSQL", "error", err)
	} else if ok {
		return true, nil
	}
	active, err := s.base.IsActive(ctx, jti)
	if err != nil {
		return false, err
	}
	if active {
		if err := s.c.Set(ctx, sessionKey(jti), []byte{'1'}, s.ttl); err != nil {
			s.log.Debug("session cache: set failed", "error", err)
		}
	}
	return active, nil
}

// Revoke revokes in PostgreSQL and drops the cached entry so the next IsActive
// re-reads (and returns false) from the durable store.
func (s *SessionCache) Revoke(ctx context.Context, jti string) error {
	if err := s.base.Revoke(ctx, jti); err != nil {
		return err
	}
	if err := s.c.Del(ctx, sessionKey(jti)); err != nil {
		s.log.Debug("session cache: del failed", "error", err)
	}
	return nil
}
