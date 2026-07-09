package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/config"
)

func TestNew_EmptyAddrReturnsNoop(t *testing.T) {
	c := New(config.Config{}, testLog())
	assert.False(t, c.Enabled())

	// noop: writes are dropped and reads always miss.
	require.NoError(t, c.Set(context.Background(), "k", []byte("v"), time.Minute))
	_, ok, err := c.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestNoopCache_DelAndClose(t *testing.T) {
	var c Cache = noopCache{}
	require.NoError(t, c.Del(context.Background(), "a", "b"))
	require.NoError(t, c.Del(context.Background())) // no keys
	require.NoError(t, c.Close())
}

func TestSessionCache_RevokeAllForUser(t *testing.T) {
	base, c := newFakeSessions(), newFakeCache()
	sc := NewSessionCache(base, c, time.Minute, testLog())
	ctx := context.Background()
	exp := time.Now().Add(time.Hour).Unix()

	require.NoError(t, sc.Create(ctx, "jti-a", "u-1", exp))
	require.NoError(t, sc.Create(ctx, "jti-b", "u-1", exp))
	require.NoError(t, sc.Create(ctx, "jti-c", "u-2", exp))

	jtis, err := sc.RevokeAllForUser(ctx, "u-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"jti-a", "jti-b"}, jtis)

	// Cache entries for the revoked sessions are dropped; the next read falls
	// through to PostgreSQL, which now reports them inactive.
	active, err := sc.IsActive(ctx, "jti-a")
	require.NoError(t, err)
	assert.False(t, active)

	// The other user's session is untouched.
	active, err = sc.IsActive(ctx, "jti-c")
	require.NoError(t, err)
	assert.True(t, active)
}

func TestSessionCache_RevokeAllForUser_BaseError(t *testing.T) {
	base := newFakeSessions()
	base.active["jti-x"] = true // present, but the revoke-all fails upstream
	sc := NewSessionCache(&errSessions{fakeSessions: base}, newFakeCache(), time.Minute, testLog())

	jtis, err := sc.RevokeAllForUser(context.Background(), "u-1")
	require.Error(t, err)
	assert.Nil(t, jtis)
}

// errSessions wraps a fakeSessions to force RevokeAllForUser to error.
type errSessions struct{ *fakeSessions }

func (e *errSessions) RevokeAllForUser(context.Context, string) ([]string, error) {
	return nil, errors.New("pg down")
}

func TestIdentityCache_CorruptCacheReloads(t *testing.T) {
	base := &fakeIdentities{ids: map[string]*auth.Identity{
		"u-1": {UserID: "u-1", Username: "alice"},
	}}
	c := newFakeCache()
	c.m[identityKey("u-1")] = []byte("not-json") // poison the cache entry
	ic := NewIdentityCache(base, c, time.Minute, testLog())

	id, err := ic.LoadIdentity(context.Background(), "u-1")
	require.NoError(t, err)
	assert.Equal(t, "alice", id.Username)
	assert.Equal(t, 1, base.loads) // fell through to PostgreSQL on unmarshal failure
}

func TestIdentityCache_GetErrorFallsThrough(t *testing.T) {
	base := &fakeIdentities{ids: map[string]*auth.Identity{
		"u-1": {UserID: "u-1", Username: "alice"},
	}}
	c := newFakeCache()
	c.getErr = errors.New("redis down")
	ic := NewIdentityCache(base, c, time.Minute, testLog())

	id, err := ic.LoadIdentity(context.Background(), "u-1")
	require.NoError(t, err) // cache error never fails the request
	assert.Equal(t, "alice", id.Username)
	assert.Equal(t, 1, base.loads)
}
