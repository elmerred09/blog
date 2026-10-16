package auth_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/elmerred09/blog/internal/auth"
)

const subject = "github:1234567"

func newKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

// validClaims are what the web server mints: 5 minutes, our issuer and
// audience, an allowed subject.
func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": auth.Issuer,
		"aud": auth.Audience,
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}
}

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestVerify(t *testing.T) {
	t.Parallel()
	pub, priv := newKey(t)
	_, otherPriv := newKey(t)
	v, err := auth.NewVerifier(pub, []string{subject})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	with := func(edit func(jwt.MapClaims)) jwt.MapClaims {
		c := validClaims()
		edit(c)
		return c
	}
	now := time.Now()

	tests := []struct {
		name  string
		token string
		want  error // nil, auth.ErrUnauthorized, or auth.ErrForbidden
	}{
		{"valid", sign(t, jwt.SigningMethodEdDSA, priv, validClaims()), nil},
		{"expired within the leeway", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["exp"] = now.Add(-10 * time.Second).Unix() })), nil},
		{"audience as a list", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["aud"] = []string{"other", auth.Audience} })), nil},

		{"empty", "", auth.ErrUnauthorized},
		{"not a JWT", "not-a-token", auth.ErrUnauthorized},
		{"expired", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Minute).Unix() })), auth.ErrUnauthorized},
		{"no exp", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { delete(c, "exp") })), auth.ErrUnauthorized},
		{"lives longer than 5 minutes", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["exp"] = now.Add(time.Hour).Unix() })), auth.ErrUnauthorized},
		{"not valid yet", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["nbf"] = now.Add(time.Minute).Unix() })), auth.ErrUnauthorized},
		{"wrong issuer", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["iss"] = "someone-else" })), auth.ErrUnauthorized},
		{"no issuer", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { delete(c, "iss") })), auth.ErrUnauthorized},
		{"wrong audience", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["aud"] = "other-api" })), auth.ErrUnauthorized},
		{"no audience", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { delete(c, "aud") })), auth.ErrUnauthorized},
		{"signed by another key", sign(t, jwt.SigningMethodEdDSA, otherPriv, validClaims()), auth.ErrUnauthorized},
		{"alg none", sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()), auth.ErrUnauthorized},
		// Algorithm confusion: an HMAC "signed" with the public key, which is
		// public. A verifier that trusts the token's alg header would accept it.
		{"HS256 keyed with the public key", sign(t, jwt.SigningMethodHS256, []byte(pub), validClaims()), auth.ErrUnauthorized},

		{"subject not allowed", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { c["sub"] = "github:999" })), auth.ErrForbidden},
		{"no subject", sign(t, jwt.SigningMethodEdDSA, priv,
			with(func(c jwt.MapClaims) { delete(c, "sub") })), auth.ErrForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := v.Verify(tc.token)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("Verify: %v", err)
				}
				if got != subject {
					t.Errorf("subject = %q, want %q", got, subject)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("Verify = %q, %v; want %v", got, err, tc.want)
			}
			if got != "" {
				t.Errorf("subject = %q on error, want empty", got)
			}
		})
	}
}

func TestNewVerifierRejectsBadConfig(t *testing.T) {
	t.Parallel()
	pub, _ := newKey(t)

	if _, err := auth.NewVerifier(pub, nil); err == nil {
		t.Error("NewVerifier with no subjects succeeded, want an error")
	}
	if _, err := auth.NewVerifier(pub, []string{""}); err == nil {
		t.Error("NewVerifier with an empty subject succeeded, want an error")
	}
	if _, err := auth.NewVerifier(ed25519.PublicKey("short"), []string{subject}); err == nil {
		t.Error("NewVerifier with a malformed key succeeded, want an error")
	}
}

func TestParsePublicKey(t *testing.T) {
	t.Parallel()
	pub, priv := newKey(t)

	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	got, err := auth.ParsePublicKey(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	if err != nil {
		t.Fatalf("ParsePublicKey: %v", err)
	}
	if !got.Equal(pub) {
		t.Error("parsed key differs from the original")
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	rsaDER, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal rsa key: %v", err)
	}

	bad := []struct {
		name string
		pem  []byte
	}{
		{"not PEM", []byte("hello")},
		{"a private key", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})},
		{"an RSA public key", pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: rsaDER})},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := auth.ParsePublicKey(tc.pem); err == nil {
				t.Error("ParsePublicKey succeeded, want an error")
			}
		})
	}
}
