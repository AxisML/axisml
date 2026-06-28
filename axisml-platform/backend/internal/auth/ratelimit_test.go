package auth

import (
	"testing"
	"time"
)

func TestRateLimiter_BurstThenRefill(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewRateLimiter(3, 1) // burst 3, 1 token/s
	l.now = func() time.Time { return now }

	// Burst of 3 succeeds, 4th is denied.
	for i := 0; i < 3; i++ {
		if !l.allow("ip") {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
	if l.allow("ip") {
		t.Fatal("4th request should be rate-limited")
	}

	// After 1s, exactly one token refills.
	now = now.Add(time.Second)
	if !l.allow("ip") {
		t.Fatal("one token should have refilled after 1s")
	}
	if l.allow("ip") {
		t.Fatal("only one token should refill in 1s")
	}
}

func TestRateLimiter_KeysAreIndependent(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewRateLimiter(1, 1)
	l.now = func() time.Time { return now }

	if !l.allow("a") {
		t.Fatal("first key should be allowed")
	}
	if !l.allow("b") {
		t.Fatal("distinct key must have its own bucket")
	}
	if l.allow("a") {
		t.Fatal("key a is exhausted")
	}
}

func TestPasswordChangeExempt(t *testing.T) {
	exempt := []string{
		"/api/v1/auth/me",
		"/api/v1/auth/logout",
		"/api/v1/auth/refresh",
		"/api/v1/users/:id/password",
	}
	for _, p := range exempt {
		if !passwordChangeExempt(p) {
			t.Errorf("%s should be exempt while a password change is owed", p)
		}
	}
	blocked := []string{"/api/v1/jobs", "/api/v1/users", "/api/v1/users/:id", "/api/v1/tenants"}
	for _, p := range blocked {
		if passwordChangeExempt(p) {
			t.Errorf("%s must be blocked until the password is changed", p)
		}
	}
}
