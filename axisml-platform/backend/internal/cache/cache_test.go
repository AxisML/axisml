package cache

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/components/platform/internal/auth"
)

func testLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeCache is an in-memory Cache with optional injected errors.
type fakeCache struct {
	m      map[string][]byte
	getErr error
	setErr error
	sets   int
	dels   int
}

func newFakeCache() *fakeCache { return &fakeCache{m: map[string][]byte{}} }

func (f *fakeCache) Get(_ context.Context, k string) ([]byte, bool, error) {
	if f.getErr != nil {
		return nil, false, f.getErr
	}
	v, ok := f.m[k]
	return v, ok, nil
}

func (f *fakeCache) Set(_ context.Context, k string, v []byte, _ time.Duration) error {
	f.sets++
	if f.setErr != nil {
		return f.setErr
	}
	f.m[k] = v
	return nil
}

func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	f.dels++
	for _, k := range keys {
		delete(f.m, k)
	}
	return nil
}

func (f *fakeCache) Enabled() bool { return true }
func (f *fakeCache) Close() error  { return nil }

// fakeSessions is an in-memory auth.SessionStore counting IsActive reads.
type fakeSessions struct {
	active        map[string]bool
	users         map[string]string // jti -> userID
	isActiveCalls int
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{active: map[string]bool{}, users: map[string]string{}}
}

func (f *fakeSessions) Create(_ context.Context, jti, userID string, _ int64) error {
	f.active[jti] = true
	f.users[jti] = userID
	return nil
}

func (f *fakeSessions) IsActive(_ context.Context, jti string) (bool, error) {
	f.isActiveCalls++
	return f.active[jti], nil
}

func (f *fakeSessions) Revoke(_ context.Context, jti string) error {
	f.active[jti] = false
	return nil
}

func (f *fakeSessions) RevokeAllForUser(_ context.Context, userID string) ([]string, error) {
	var jtis []string
	for jti, uid := range f.users {
		if uid == userID && f.active[jti] {
			f.active[jti] = false
			jtis = append(jtis, jti)
		}
	}
	return jtis, nil
}

func TestSessionCache_WriteThroughThenHit(t *testing.T) {
	base, c := newFakeSessions(), newFakeCache()
	sc := NewSessionCache(base, c, time.Minute, testLog())
	ctx := context.Background()

	require.NoError(t, sc.Create(ctx, "jti-1", "u-1", time.Now().Add(time.Hour).Unix()))

	// First read is a cache hit (write-through populated it): PG untouched.
	active, err := sc.IsActive(ctx, "jti-1")
	require.NoError(t, err)
	assert.True(t, active)
	assert.Equal(t, 0, base.isActiveCalls)
}

func TestSessionCache_MissRepopulates(t *testing.T) {
	base, c := newFakeSessions(), newFakeCache()
	base.active["jti-1"] = true // present in PG, not yet cached
	sc := NewSessionCache(base, c, time.Minute, testLog())
	ctx := context.Background()

	active, err := sc.IsActive(ctx, "jti-1")
	require.NoError(t, err)
	assert.True(t, active)
	assert.Equal(t, 1, base.isActiveCalls) // one fall-through

	// Now cached: a second read does not hit PG again.
	active, err = sc.IsActive(ctx, "jti-1")
	require.NoError(t, err)
	assert.True(t, active)
	assert.Equal(t, 1, base.isActiveCalls)
}

func TestSessionCache_RevokeDropsEntry(t *testing.T) {
	base, c := newFakeSessions(), newFakeCache()
	sc := NewSessionCache(base, c, time.Minute, testLog())
	ctx := context.Background()
	require.NoError(t, sc.Create(ctx, "jti-1", "u-1", time.Now().Add(time.Hour).Unix()))

	require.NoError(t, sc.Revoke(ctx, "jti-1"))

	// Cache entry gone → falls through to PG, which now reports inactive.
	active, err := sc.IsActive(ctx, "jti-1")
	require.NoError(t, err)
	assert.False(t, active)
	assert.Equal(t, 1, base.isActiveCalls)
}

func TestSessionCache_GetErrorFallsThrough(t *testing.T) {
	base, c := newFakeSessions(), newFakeCache()
	base.active["jti-1"] = true
	c.getErr = errors.New("redis down")
	sc := NewSessionCache(base, c, time.Minute, testLog())

	active, err := sc.IsActive(context.Background(), "jti-1")
	require.NoError(t, err) // cache error never fails the request
	assert.True(t, active)
	assert.Equal(t, 1, base.isActiveCalls)
}

// fakeIdentities is an in-memory auth.IdentityStore counting loads.
type fakeIdentities struct {
	ids   map[string]*auth.Identity
	loads int
}

func (f *fakeIdentities) LoadIdentity(_ context.Context, userID string) (*auth.Identity, error) {
	f.loads++
	id, ok := f.ids[userID]
	if !ok {
		return nil, errors.New("not found")
	}
	return id, nil
}

func TestIdentityCache_MissThenHitThenInvalidate(t *testing.T) {
	base := &fakeIdentities{ids: map[string]*auth.Identity{
		"u-1": {UserID: "u-1", Username: "alice", Bindings: map[string]auth.Role{"t1": "tenant-admin"}},
	}}
	c := newFakeCache()
	ic := NewIdentityCache(base, c, time.Minute, testLog())
	ctx := context.Background()

	// Miss → PG load.
	id, err := ic.LoadIdentity(ctx, "u-1")
	require.NoError(t, err)
	assert.Equal(t, "alice", id.Username)
	assert.Equal(t, auth.Role("tenant-admin"), id.Bindings["t1"])
	assert.Equal(t, 1, base.loads)

	// Hit → no further PG load, bindings round-trip through JSON.
	id, err = ic.LoadIdentity(ctx, "u-1")
	require.NoError(t, err)
	assert.Equal(t, auth.Role("tenant-admin"), id.Bindings["t1"])
	assert.Equal(t, 1, base.loads)

	// Invalidate forces a reload.
	ic.Invalidate(ctx, "u-1")
	_, err = ic.LoadIdentity(ctx, "u-1")
	require.NoError(t, err)
	assert.Equal(t, 2, base.loads)
}

func TestIdentityCache_ErrorsNotCached(t *testing.T) {
	base := &fakeIdentities{ids: map[string]*auth.Identity{}}
	c := newFakeCache()
	ic := NewIdentityCache(base, c, time.Minute, testLog())
	ctx := context.Background()

	_, err := ic.LoadIdentity(ctx, "ghost")
	require.Error(t, err)
	_, err = ic.LoadIdentity(ctx, "ghost")
	require.Error(t, err)
	assert.Equal(t, 2, base.loads) // each call hit PG; nothing cached
	assert.Equal(t, 0, c.sets)
}

func TestNoopCache_AlwaysMisses(t *testing.T) {
	var c Cache = noopCache{}
	require.NoError(t, c.Set(context.Background(), "k", []byte("v"), time.Minute))
	_, ok, err := c.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.False(t, c.Enabled())
}
