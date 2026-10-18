package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// limitBody caps the request body. The handler that reads it turns a tripped
// cap into a 413.
func limitBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

// decodeJSON reads one JSON object into dst. Unknown fields are rejected.
// It writes the error response and returns false when the body is unusable.
func decodeJSON(c *gin.Context, dst any) bool {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			writeError(c, http.StatusRequestEntityTooLarge, codeTooLarge, "request body is too large")
			return false
		}
		writeError(c, http.StatusBadRequest, codeInvalidRequest, "invalid JSON")
		return false
	}
	// A second value would be a second JSON document. One object is the body.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, codeInvalidRequest, "invalid JSON")
		return false
	}
	return true
}

// Responses encode times in UTC: call .UTC() on a time.Time field, or utcPtr
// on a nullable one.
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
