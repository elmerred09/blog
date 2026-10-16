// Package auth verifies the short-lived tokens the web server signs for each
// admin API call. Go holds only the Ed25519 public key, so it can check a
// token but never mint one.
package auth

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer   = "blog-web"
	Audience = "blog-api"
	// MaxTTL caps how far in the future a token may expire, so a token that
	// leaks, or a signer bug, can't grant long-lived access.
	MaxTTL = 5 * time.Minute
	// leeway absorbs clock skew between the web server and the API.
	leeway = 30 * time.Second
)

var (
	// ErrUnauthorized means no usable token: missing, malformed, badly
	// signed, expired, or for another issuer or audience.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden means a valid token whose subject isn't allowed.
	ErrForbidden = errors.New("forbidden")
)

type Verifier struct {
	key      ed25519.PublicKey
	subjects map[string]bool
	parser   *jwt.Parser
}

// NewVerifier returns a Verifier that accepts tokens signed by key's private
// half for one of subjects (for example "github:1234567").
func NewVerifier(key ed25519.PublicKey, subjects []string) (*Verifier, error) {
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("ed25519 public key is %d bytes, want %d", len(key), ed25519.PublicKeySize)
	}
	if len(subjects) == 0 {
		return nil, errors.New("no allowed subjects")
	}
	allowed := make(map[string]bool, len(subjects))
	for _, s := range subjects {
		if s == "" {
			return nil, errors.New("empty allowed subject")
		}
		allowed[s] = true
	}
	return &Verifier{
		key:      key,
		subjects: allowed,
		parser: jwt.NewParser(
			// The algorithm comes from here, never from the token's header:
			// that is what stops "alg: none" and HS256-with-the-public-key.
			jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
			jwt.WithIssuer(Issuer),
			jwt.WithAudience(Audience),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(leeway),
		),
	}, nil
}

// Verify checks a token and returns its subject. Errors wrap ErrUnauthorized
// or ErrForbidden, with the reason for logs; don't send the reason to clients.
func (v *Verifier) Verify(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("%w: no token", ErrUnauthorized)
	}
	var claims jwt.RegisteredClaims
	_, err := v.parser.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return v.key, nil })
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnauthorized, err)
	}
	if ttl := time.Until(claims.ExpiresAt.Time); ttl > MaxTTL+leeway {
		return "", fmt.Errorf("%w: token expires in %v, more than %v", ErrUnauthorized, ttl.Round(time.Second), MaxTTL)
	}
	if !v.subjects[claims.Subject] {
		return "", fmt.Errorf("%w: subject %q is not allowed", ErrForbidden, claims.Subject)
	}
	return claims.Subject, nil
}

// ParsePublicKey reads an Ed25519 public key from PEM ("PUBLIC KEY", the
// SPKI form that openssl and jose's exportSPKI write).
func ParsePublicKey(pemBytes []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("no PUBLIC KEY PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is %T, want ed25519", key)
	}
	return pub, nil
}
