package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/elmerred09/blog/internal/auth"
)

const testSubject = "github:1234567"

func TestRequireAdmin(t *testing.T) {
	t.Parallel()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	v, err := auth.NewVerifier(pub, []string{testSubject})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	token := func(sub string, exp time.Time) string {
		s, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
			"iss": auth.Issuer, "aud": auth.Audience, "sub": sub, "exp": exp.Unix(),
		}).SignedString(priv)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return s
	}
	valid := token(testSubject, time.Now().Add(time.Minute))

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantCode   string
		wantLog    string // substring of the logged error
	}{
		{"valid", "Bearer " + valid, http.StatusOK, "", ""},
		{"scheme is case-insensitive", "bearer " + valid, http.StatusOK, "", ""},
		{"no header", "", http.StatusUnauthorized, codeUnauthorized, "no token"},
		{"other scheme", "Basic dXNlcjpwYXNz", http.StatusUnauthorized, codeUnauthorized, "no token"},
		{"bearer without a token", "Bearer ", http.StatusUnauthorized, codeUnauthorized, "no token"},
		{"expired", "Bearer " + token(testSubject, time.Now().Add(-time.Hour)), http.StatusUnauthorized, codeUnauthorized, "expired"},
		{"subject not allowed", "Bearer " + token("github:999", time.Now().Add(time.Minute)), http.StatusForbidden, codeForbidden, "github:999"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			log, buf := testLogger()
			r := newEngine(Deps{Logger: log})
			r.GET("/admin-probe", requireAdmin(v), func(c *gin.Context) {
				c.String(http.StatusOK, adminSubject(c))
			})

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/admin-probe", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := serve(t, r, req)

			if tc.wantStatus == http.StatusOK {
				if rec.Code != http.StatusOK || rec.Body.String() != testSubject {
					t.Fatalf("got %d %q, want 200 with the subject", rec.Code, rec.Body)
				}
				return
			}
			wantError(t, rec, tc.wantStatus, tc.wantCode)
			if tc.wantStatus == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
			}
			// The reason goes to the log, never to the client.
			if strings.Contains(rec.Body.String(), tc.wantLog) {
				t.Errorf("body %s leaks the reason %q", rec.Body, tc.wantLog)
			}
			line := requestLine(t, buf)
			if logged, _ := line["error"].(string); !strings.Contains(logged, tc.wantLog) || line["level"] != "WARN" {
				t.Errorf("request log = %v, want a WARN with an error containing %q", line, tc.wantLog)
			}
		})
	}
}
