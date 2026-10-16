package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/elmerred09/blog/internal/auth"
)

const adminSubjectKey = "admin_subject"

// requireAdmin lets a request through only with a valid admin token in
// "Authorization: Bearer <token>". The verifier's reason is logged with the
// request; the client gets only the code.
func requireAdmin(v *auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := v.Verify(bearerToken(c.GetHeader("Authorization")))
		if err != nil {
			_ = c.Error(err)
			if errors.Is(err, auth.ErrForbidden) {
				writeError(c, http.StatusForbidden, codeForbidden, "not allowed")
				return
			}
			c.Header("WWW-Authenticate", "Bearer")
			writeError(c, http.StatusUnauthorized, codeUnauthorized, "authentication required")
			return
		}
		c.Set(adminSubjectKey, sub)
		c.Next()
	}
}

// adminSubject returns the subject requireAdmin verified for this request.
func adminSubject(c *gin.Context) string {
	return c.GetString(adminSubjectKey)
}

// bearerToken returns the token from an Authorization header, or "" if the
// scheme isn't Bearer. The scheme is case-insensitive (RFC 9110).
func bearerToken(header string) string {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
