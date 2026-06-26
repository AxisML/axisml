package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Audience is the control-plane login audience (auth.md §5).
const Audience = "axisml-platform"

// Claims is the login JWT payload.
type Claims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
}

// Signer issues and verifies RS256 login JWTs and publishes the public JWKS.
type Signer struct {
	priv *rsa.PrivateKey
	kid  string
	ttl  time.Duration
}

// NewSigner parses an RSA private key from PEM (PKCS#1 or PKCS#8). When pemKey
// is empty it generates an ephemeral 2048-bit key — acceptable for a single
// dev replica, but production should inject a stable key so tokens survive
// restarts and JWKS stays consistent across replicas. The JWKS kid is derived
// from the key as its RFC 7638 thumbprint, so replicas sharing a key agree
// automatically and rotation is implicit.
func NewSigner(pemKey string, ttl time.Duration) (*Signer, error) {
	var priv *rsa.PrivateKey
	if pemKey == "" {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate jwt key: %w", err)
		}
		priv = k
	} else {
		block, _ := pem.Decode([]byte(pemKey))
		if block == nil {
			return nil, fmt.Errorf("jwt key: invalid PEM")
		}
		if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			priv = k
		} else if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			rk, ok := k.(*rsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("jwt key: not an RSA key")
			}
			priv = rk
		} else {
			return nil, fmt.Errorf("jwt key: parse PKCS1/PKCS8: %w", err)
		}
	}
	return &Signer{priv: priv, kid: thumbprint(&priv.PublicKey), ttl: ttl}, nil
}

// thumbprint returns the RFC 7638 JWK thumbprint of an RSA public key: the
// base64url(SHA-256) of the canonical JWK with members in lexicographic order
// (e, kty, n). This is the stable key id published as the JWKS kid.
func thumbprint(pub *rsa.PublicKey) string {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	jwk := `{"e":"` + e + `","kty":"RSA","n":"` + n + `"}`
	sum := sha256.Sum256([]byte(jwk))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// TTL is the configured login-token lifetime.
func (s *Signer) TTL() time.Duration { return s.ttl }

// Issue mints a login JWT for the user. Returns the signed token and its expiry.
func (s *Signer) Issue(userID, username, jti string, now time.Time) (string, time.Time, error) {
	exp := now.Add(s.ttl)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			Audience:  jwt.ClaimStrings{Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		Username: username,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = s.kid
	signed, err := tok.SignedString(s.priv)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// Verify validates a token's signature, audience and expiry and returns its
// claims.
func (s *Signer) Verify(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return &s.priv.PublicKey, nil
	}, jwt.WithAudience(Audience))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// JWKS returns the public key set served at /.well-known/jwks.json.
func (s *Signer) JWKS() map[string]any {
	pub := s.priv.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	return map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": s.kid, "n": n, "e": e,
		}},
	}
}
