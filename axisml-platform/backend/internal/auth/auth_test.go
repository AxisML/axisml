package auth

import (
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
	signer, err := NewSigner("", "kid-1", time.Hour)
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
	signer, _ := NewSigner("", "kid-1", time.Hour)
	jwks := signer.JWKS()
	keys, ok := jwks["keys"].([]map[string]any)
	if !ok || len(keys) != 1 || keys[0]["kid"] != "kid-1" || keys[0]["kty"] != "RSA" {
		t.Fatalf("unexpected jwks: %+v", jwks)
	}
}
