package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
	maxRequestIDLen = 64
)

// requestID reuses the caller's X-Request-ID when it is safe to log, otherwise
// generates one, and echoes it in the response.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !validRequestID(id) {
			id = uuid.NewString()
		}
		c.Set(requestIDKey, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// validRequestID accepts short ids of [A-Za-z0-9._-], so a client can't inject
// newlines or huge values into the logs.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, r := range id {
		switch {
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9', r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// requestLogger writes one line per request after it completes: 5xx at Error,
// 4xx at Warn, everything else at Info. The route is the registered pattern
// (/api/posts/:slug), so lines group by endpoint rather than by URL.
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			slog.Int("bytes", max(c.Writer.Size(), 0)),
			slog.String("request_id", c.GetString(requestIDKey)),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.String()))
		}
		log.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}

// recovery turns a panic into a JSON 500 and logs the panic value and stack.
// http.ErrAbortHandler is re-panicked: net/http uses it to abort a response
// on purpose.
func recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			log.LogAttrs(c.Request.Context(), slog.LevelError, "panic",
				slog.String("panic", fmt.Sprint(rec)),
				slog.String("stack", string(debug.Stack())),
				slog.String("request_id", c.GetString(requestIDKey)),
			)
			if c.Writer.Written() {
				c.Abort()
				return
			}
			internalError(c, fmt.Errorf("panic: %v", rec))
		}()
		c.Next()
	}
}
