package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

func TestRoleOrdering(t *testing.T) {
	if !RoleSystemAdmin.AtLeast(RoleTenantAdmin) || !RoleTenantAdmin.AtLeast(RoleUser) {
		t.Fatal("role ordering broken")
	}
	if RoleUser.AtLeast(RoleTenantAdmin) {
		t.Fatal("user must not satisfy tenant-admin")
	}
}

func TestIdentityHasTenantRole(t *testing.T) {
	id := &Identity{Bindings: map[string]Role{"acme": RoleUser}}
	if !id.HasTenantRole("acme", RoleUser) {
		t.Fatal("user should satisfy user")
	}
	if id.HasTenantRole("acme", RoleTenantAdmin) {
		t.Fatal("user must not satisfy tenant-admin")
	}
	if id.HasTenantRole("other", RoleUser) {
		t.Fatal("no binding must not satisfy")
	}

	admin := &Identity{IsSystemAdmin: true}
	if !admin.HasTenantRole("any", RoleTenantAdmin) {
		t.Fatal("system-admin must short-circuit any tenant role")
	}
}

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "s3cret-pw") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	signer, err := NewSigner("", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok, exp, err := signer.Issue("user-id", "alice", "jti-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !exp.After(time.Now()) {
		t.Fatal("expiry not in the future")
	}
	claims, err := signer.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != "user-id" || claims.Username != "alice" || claims.ID != "jti-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := signer.Verify(tok + "tamper"); err == nil {
		t.Fatal("tampered token verified")
	}
}

func TestJWKS(t *testing.T) {
	signer, _ := NewSigner("", time.Hour)
	jwks := signer.JWKS()
	keys, ok := jwks["keys"].([]map[string]any)
	if !ok || len(keys) != 1 || keys[0]["kty"] != "RSA" {
		t.Fatalf("unexpected jwks: %+v", jwks)
	}
	// kid is the derived thumbprint and must match what tokens are signed with.
	if keys[0]["kid"] == "" {
		t.Fatal("jwks kid is empty")
	}
}

// TestDerivedKidStable verifies the kid is a deterministic function of the key:
// the same PEM yields the same kid (so replicas sharing a key agree), and a
// different key yields a different kid.
func TestDerivedKidStable(t *testing.T) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k),
	}))

	kid := func(pemKey string) string {
		s, err := NewSigner(pemKey, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return s.JWKS()["keys"].([]map[string]any)[0]["kid"].(string)
	}

	a, b := kid(pemKey), kid(pemKey)
	if a == "" || a != b {
		t.Fatalf("kid not stable for same key: %q vs %q", a, b)
	}
	if c := kid(""); c == a {
		t.Fatal("a different key should derive a different kid")
	}
}
